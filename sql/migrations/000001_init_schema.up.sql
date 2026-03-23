-- 初始迁移：创建所有表结构
-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    email TEXT DEFAULT '',
    role TEXT DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME,
    last_login_ip TEXT,
    login_count INTEGER DEFAULT 0,
    avatar TEXT DEFAULT '',
    nickname TEXT DEFAULT '',
    region TEXT DEFAULT '',
    configs TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- VM 表
CREATE TABLE IF NOT EXISTS vms (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT NOT NULL,
    password TEXT DEFAULT '',
    key_path TEXT DEFAULT '',
    status TEXT DEFAULT 'stopped',
    description TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vms_name ON vms(name);
CREATE INDEX IF NOT EXISTS idx_vms_status ON vms(status);

-- 虚拟机组表
CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    vms TEXT DEFAULT '[]',
    owner TEXT DEFAULT '',
    is_shared INTEGER DEFAULT 0,
    shared_with TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);
CREATE INDEX IF NOT EXISTS idx_groups_owner ON groups(owner);
CREATE INDEX IF NOT EXISTS idx_groups_is_shared ON groups(is_shared);

-- 进程表
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

-- 登录日志表
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

-- 命令模板表
CREATE TABLE IF NOT EXISTS command_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    command TEXT NOT NULL,
    variables TEXT DEFAULT '[]',
    category TEXT DEFAULT '',
    is_shared INTEGER DEFAULT 0,
    created_by TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_templates_name ON command_templates(name);
CREATE INDEX IF NOT EXISTS idx_templates_category ON command_templates(category);
CREATE INDEX IF NOT EXISTS idx_templates_created_by ON command_templates(created_by);
CREATE INDEX IF NOT EXISTS idx_templates_is_shared ON command_templates(is_shared);

-- 命令历史表
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
