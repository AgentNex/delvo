package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// OpenDatabase creates or opens the SQLite database and executes migrations.
func OpenDatabase(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=temp_store(MEMORY)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// SQLite performs best with a single connection for write safety
	db.SetMaxOpenConns(1)

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return db, nil
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("execute schema sql: %w", err)
	}

	// Ensure schema version recorded
	_, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version) VALUES (?)`, CurrentSchemaVersion)
	return err
}
