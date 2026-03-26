-- 添加定时任务表
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    command TEXT NOT NULL,
    vm_names TEXT DEFAULT '[]',
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

-- 定时任务执行历史表
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
