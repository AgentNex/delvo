package index

const CurrentSchemaVersion = 1

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS entries (
    path TEXT PRIMARY KEY,
    parent_path TEXT NOT NULL,
    name TEXT NOT NULL,
    extension TEXT NOT NULL,
    is_dir INTEGER NOT NULL,
    size INTEGER NOT NULL,
    allocated_size INTEGER NOT NULL,
    mod_time INTEGER NOT NULL,
    is_hidden INTEGER NOT NULL,
    is_symlink INTEGER NOT NULL,
    symlink_target TEXT,
    dev INTEGER,
    ino INTEGER,
    scan_status TEXT,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_entries_parent ON entries(parent_path);
CREATE INDEX IF NOT EXISTS idx_entries_name ON entries(name);
CREATE INDEX IF NOT EXISTS idx_entries_ext ON entries(extension);
CREATE INDEX IF NOT EXISTS idx_entries_size ON entries(size);
CREATE INDEX IF NOT EXISTS idx_entries_mod_time ON entries(mod_time);
CREATE INDEX IF NOT EXISTS idx_entries_is_dir ON entries(is_dir);

CREATE TABLE IF NOT EXISTS scan_meta (
    root TEXT PRIMARY KEY,
    last_scan INTEGER NOT NULL,
    item_count INTEGER NOT NULL,
    total_size INTEGER NOT NULL,
    status TEXT NOT NULL
);
`
