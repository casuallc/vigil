-- Command history table schema
CREATE TABLE IF NOT EXISTS command_history (
    id TEXT PRIMARY KEY,
    vm_name TEXT NOT NULL,
    command TEXT NOT NULL,
    executed_by TEXT DEFAULT '',
    executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'success',
    duration_ms INTEGER DEFAULT 0,
    output TEXT DEFAULT '',
    error TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_history_vm ON command_history(vm_name);
CREATE INDEX IF NOT EXISTS idx_history_executed_at ON command_history(executed_at);
CREATE INDEX IF NOT EXISTS idx_history_executed_by ON command_history(executed_by);
CREATE INDEX IF NOT EXISTS idx_history_status ON command_history(status);
