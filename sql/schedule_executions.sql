-- Schedule executions table schema
CREATE TABLE IF NOT EXISTS schedule_executions (
    id TEXT PRIMARY KEY,
    schedule_id TEXT NOT NULL,
    triggered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    status TEXT DEFAULT 'running',
    results TEXT DEFAULT '{}',
    trigger_type TEXT DEFAULT 'auto',
    FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_schedule_executions_schedule_id ON schedule_executions(schedule_id);
CREATE INDEX IF NOT EXISTS idx_schedule_executions_triggered_at ON schedule_executions(triggered_at);
CREATE INDEX IF NOT EXISTS idx_schedule_executions_status ON schedule_executions(status);
