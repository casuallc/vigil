-- 回滚初始迁移：删除所有表
DROP TABLE IF EXISTS command_history;
DROP TABLE IF EXISTS command_templates;
DROP TABLE IF EXISTS login_logs;
DROP TABLE IF EXISTS procs;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS vms;
DROP TABLE IF EXISTS users;
