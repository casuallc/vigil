-- Login Logs table schema
CREATE TABLE IF NOT EXISTS login_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    user_id TEXT,
    client_ip TEXT,
    user_agent TEXT,
    device_info TEXT,
    login_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'success'
);

CREATE INDEX IF NOT EXISTS idx_login_logs_username ON login_logs(username);
CREATE INDEX IF NOT EXISTS idx_login_logs_login_time ON login_logs(login_time);
