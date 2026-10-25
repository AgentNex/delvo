package index

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/AgentNex/delvo/internal/scanner"
)

// ScanMeta stores scan statistics for a configured root.
type ScanMeta struct {
	Root      string    `json:"root"`
	LastScan  time.Time `json:"last_scan"`
	ItemCount int64     `json:"item_count"`
	TotalSize int64     `json:"total_size"`
	Status    string    `json:"status"`
}

// BatchUpsert writes a batch of entries into the database within a transaction.
func (idx *Index) BatchUpsert(ctx context.Context, entries []scanner.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO entries (
			path, parent_path, name, extension, is_dir, size, allocated_size,
			mod_time, is_hidden, is_symlink, symlink_target, dev, ino, scan_status, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			parent_path = excluded.parent_path,
			name = excluded.name,
			extension = excluded.extension,
			is_dir = excluded.is_dir,
			size = excluded.size,
			allocated_size = excluded.allocated_size,
			mod_time = excluded.mod_time,
			is_hidden = excluded.is_hidden,
			is_symlink = excluded.is_symlink,
			symlink_target = excluded.symlink_target,
			dev = excluded.dev,
			ino = excluded.ino,
			scan_status = excluded.scan_status,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return fmt.Errorf("prepare upsert stmt: %w", err)
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, e := range entries {
		isDir := 0
		if e.IsDir {
			isDir = 1
		}
		isHidden := 0
		if e.IsHidden {
			isHidden = 1
		}
		isSymlink := 0
		if e.IsSymlink {
			isSymlink = 1
		}

		_, err := stmt.ExecContext(ctx,
			e.Path,
			e.ParentPath,
			e.Name,
			e.Extension,
			isDir,
			e.Size,
			e.AllocatedSize,
			e.ModTime.Unix(),
			isHidden,
			isSymlink,
			e.SymlinkTarget,
			e.Dev,
			e.Ino,
			e.ScanStatus,
			now,
		)
		if err != nil {
			return fmt.Errorf("exec upsert: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateDirSizes updates the calculated apparent sizes for directories.
func (idx *Index) UpdateDirSizes(ctx context.Context, dirSizes map[string]int64) error {
	if len(dirSizes) == 0 {
		return nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `UPDATE entries SET size = ? WHERE path = ? AND is_dir = 1`)
	if err != nil {
		return fmt.Errorf("prepare update dir size: %w", err)
	}
	defer stmt.Close()

	for path, size := range dirSizes {
		if _, err := stmt.ExecContext(ctx, size, path); err != nil {
			return fmt.Errorf("exec update dir size for %s: %w", path, err)
		}
	}

	return tx.Commit()
}

// DeleteEntry removes an entry and its sub-entries from the index.
func (idx *Index) DeleteEntry(ctx context.Context, path string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	_, err := idx.db.ExecContext(ctx, `DELETE FROM entries WHERE path = ? OR path LIKE ?`, path, path+"/%")
	return err
}

// GetEntry retrieves a single entry by exact path.
func (idx *Index) GetEntry(ctx context.Context, path string) (*scanner.Entry, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	row := idx.db.QueryRowContext(ctx, `
		SELECT path, parent_path, name, extension, is_dir, size, allocated_size,
		       mod_time, is_hidden, is_symlink, symlink_target, dev, ino, scan_status
		FROM entries WHERE path = ?
	`, path)

	var (
		e                      scanner.Entry
		isDir, isHidden, isSym int
		modUnix                int64
	)

	err := row.Scan(
		&e.Path, &e.ParentPath, &e.Name, &e.Extension,
		&isDir, &e.Size, &e.AllocatedSize,
		&modUnix, &isHidden, &isSym, &e.SymlinkTarget,
		&e.Dev, &e.Ino, &e.ScanStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	e.IsDir = isDir == 1
	e.IsHidden = isHidden == 1
	e.IsSymlink = isSym == 1
	e.ModTime = time.Unix(modUnix, 0)

	return &e, nil
}

// GetEntriesInDir retrieves direct children of a directory.
func (idx *Index) GetEntriesInDir(ctx context.Context, parentPath string) ([]scanner.Entry, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	rows, err := idx.db.QueryContext(ctx, `
		SELECT path, parent_path, name, extension, is_dir, size, allocated_size,
		       mod_time, is_hidden, is_symlink, symlink_target, dev, ino, scan_status
		FROM entries
		WHERE parent_path = ?
		ORDER BY is_dir DESC, name COLLATE NOCASE ASC
	`, parentPath)
	if err != nil {
		return nil, fmt.Errorf("query dir entries: %w", err)
	}
	defer rows.Close()

	var entries []scanner.Entry
	for rows.Next() {
		var (
			e                      scanner.Entry
			isDir, isHidden, isSym int
			modUnix                int64
		)
		if err := rows.Scan(
			&e.Path, &e.ParentPath, &e.Name, &e.Extension,
			&isDir, &e.Size, &e.AllocatedSize,
			&modUnix, &isHidden, &isSym, &e.SymlinkTarget,
			&e.Dev, &e.Ino, &e.ScanStatus,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		e.IsDir = isDir == 1
		e.IsHidden = isHidden == 1
		e.IsSymlink = isSym == 1
		e.ModTime = time.Unix(modUnix, 0)
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// GetAllEntries retrieves all entries indexed.
func (idx *Index) GetAllEntries(ctx context.Context) ([]scanner.Entry, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	rows, err := idx.db.QueryContext(ctx, `
		SELECT path, parent_path, name, extension, is_dir, size, allocated_size,
		       mod_time, is_hidden, is_symlink, symlink_target, dev, ino, scan_status
		FROM entries
		ORDER BY is_dir DESC, name COLLATE NOCASE ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query all entries: %w", err)
	}
	defer rows.Close()

	var entries []scanner.Entry
	for rows.Next() {
		var (
			e                      scanner.Entry
			isDir, isHidden, isSym int
			modUnix                int64
		)
		if err := rows.Scan(
			&e.Path, &e.ParentPath, &e.Name, &e.Extension,
			&isDir, &e.Size, &e.AllocatedSize,
			&modUnix, &isHidden, &isSym, &e.SymlinkTarget,
			&e.Dev, &e.Ino, &e.ScanStatus,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		e.IsDir = isDir == 1
		e.IsHidden = isHidden == 1
		e.IsSymlink = isSym == 1
		e.ModTime = time.Unix(modUnix, 0)
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// UpdateScanMeta updates the metadata for a root.
func (idx *Index) UpdateScanMeta(ctx context.Context, root string, count int64, size int64, status string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	_, err := idx.db.ExecContext(ctx, `
		INSERT INTO scan_meta (root, last_scan, item_count, total_size, status)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(root) DO UPDATE SET
			last_scan = excluded.last_scan,
			item_count = excluded.item_count,
			total_size = excluded.total_size,
			status = excluded.status
	`, root, time.Now().Unix(), count, size, status)
	return err
}

// Rebuild drops and recreates tables to clear the index completely.
func (idx *Index) Rebuild(ctx context.Context) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	_, err := idx.db.ExecContext(ctx, `
		DROP TABLE IF EXISTS entries;
		DROP TABLE IF EXISTS scan_meta;
	`)
	if err != nil {
		return fmt.Errorf("drop tables: %w", err)
	}

	return initSchema(idx.db)
}
