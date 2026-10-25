package watcher

import (
	"context"
	"fmt"
	"time"
)

// ChangeType represents the classification of filesystem mutation.
type ChangeType int

const (
	ChangeCreate ChangeType = iota
	ChangeWrite
	ChangeRemove
	ChangeRename
)

func (c ChangeType) String() string {
	switch c {
	case ChangeCreate:
		return "CREATE"
	case ChangeWrite:
		return "WRITE"
	case ChangeRemove:
		return "REMOVE"
	case ChangeRename:
		return "RENAME"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", c)
	}
}

// ChangeEvent encapsulates a single normalized filesystem event.
type ChangeEvent struct {
	Path      string
	Type      ChangeType
	Timestamp time.Time
}

// ChangeSource is the pluggable interface for filesystem event backends.
type ChangeSource interface {
	Start(ctx context.Context) (<-chan ChangeEvent, <-chan error, error)
	AddWatch(path string) error
	RemoveWatch(path string) error
	Close() error
}
