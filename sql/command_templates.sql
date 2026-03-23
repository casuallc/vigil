-- Command templates table schema
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
