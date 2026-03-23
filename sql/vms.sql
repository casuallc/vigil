-- VMs table schema
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

-- Groups table schema
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
