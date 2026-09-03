-- Proxy instances table schema
CREATE TABLE IF NOT EXISTS proxy_instances (
    name TEXT PRIMARY KEY,
    origin TEXT NOT NULL,           -- config | api
    config_json TEXT NOT NULL,      -- proxy.InstanceConfig JSON
    desired_state TEXT NOT NULL,    -- running | stopped
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
