-- Schedules table schema
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    command TEXT NOT NULL,
    vm_names TEXT NOT NULL DEFAULT '[]',
    cron TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    timeout INTEGER DEFAULT 300,
    created_by TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_run_at DATETIME,
    last_run_status TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_schedules_name ON schedules(name);
CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_schedules_created_by ON schedules(created_by);
CREATE INDEX IF NOT EXISTS idx_schedules_last_run_at ON schedules(last_run_at);
