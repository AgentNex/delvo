package index

import (
	"database/sql"
	"fmt"
	"sync"
)

// Index provides thread-safe access to Delvo's local SQLite cache.
type Index struct {
	db *sql.DB
	mu sync.RWMutex
}

// New creates or connects to a local SQLite index at dbPath.
func New(dbPath string) (*Index, error) {
	db, err := OpenDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &Index{
		db: db,
	}, nil
}

// Close closes the underlying SQLite database connection.
func (idx *Index) Close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.db != nil {
		return idx.db.Close()
	}
	return nil
}
