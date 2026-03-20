-- Procs table schema
CREATE TABLE IF NOT EXISTS procs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_procs_namespace ON procs(namespace);
CREATE INDEX IF NOT EXISTS idx_procs_name ON procs(name);
