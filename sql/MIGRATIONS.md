# 数据库迁移系统

本项目使用 [golang-migrate/migrate](https://github.com/golang-migrate/migrate) 进行数据库版本管理。

## 目录结构

```
sql/
├── migrations/                 # 迁移文件目录
│   ├── 000001_init_schema.up.sql    # 初始 schema 创建
│   ├── 000001_init_schema.down.sql  # 初始 schema 回滚
│   └── ...                          # 后续迁移文件
├── migrate.go                  # 迁移管理器
├── load.go                     # SQL 加载工具
└── *.sql                       # 旧版 schema 文件（保留兼容）
```

## 迁移文件命名规范

```
{version}_{title}.up.sql      # 升级迁移
{version}_{title}.down.sql    # 回滚迁移
```

示例：
- `000001_init_schema.up.sql` - 创建初始表结构
- `000001_init_schema.down.sql` - 删除所有表

## 在代码中使用

### 自动迁移（推荐）

数据库初始化时会自动执行所有待执行的迁移：

```go
import (
    dbsql "github.com/casuallc/vigil/sql"
)

// 方法1: 从已打开的数据库连接
import "database/sql"
db, _ := sql.Open("sqlite", "path/to/db.sqlite")
dbsql.InitAndMigrate(db)

// 方法2: 从数据库路径（自动打开并关闭）
dbsql.InitAndMigrateWithPath("path/to/db.sqlite")
```

### 手动控制迁移

```go
import (
    "database/sql"
    dbsql "github.com/casuallc/vigil/sql"
)

db, _ := sql.Open("sqlite", "path/to/db.sqlite")

// 创建迁移管理器
mm, err := dbsql.NewMigrationManager(db)
if err != nil {
    log.Fatal(err)
}
defer mm.Close()

// 执行所有待执行的迁移
err = mm.Up()

// 迁移到指定版本
err = mm.UpTo(5)

// 回滚一个版本
err = mm.Down()

// 回滚到指定版本
err = mm.DownTo(3)

// 删除所有表
err = mm.Drop()

// 获取当前版本
version, dirty, err := mm.Version()

// 强制设置版本（修复脏状态）
err = mm.Force(5)
```

## 创建新的迁移

### 1. 手动创建

创建两个文件：

```bash
# 升级迁移
touch sql/migrations/000002_add_new_column.up.sql

# 回滚迁移
touch sql/migrations/000002_add_new_column.down.sql
```

### 2. 使用 migrate CLI（可选）

安装 migrate CLI：

```bash
# Windows (使用 scoop)
scoop install migrate

# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

创建新迁移：

```bash
migrate create -ext sql -dir sql/migrations -seq add_user_preferences
```

这会创建：
- `sql/migrations/000002_add_user_preferences.up.sql`
- `sql/migrations/000002_add_user_preferences.down.sql`

## 迁移示例

### 添加新列

`000002_add_profile_fields.up.sql`:
```sql
ALTER TABLE users ADD COLUMN avatar TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN nickname TEXT DEFAULT '';
```

`000002_add_profile_fields.down.sql`:
```sql
-- SQLite 不支持 DROP COLUMN，需要重建表
-- 详细实现见相关文档
```

### 创建新表

`000003_add_audit_logs.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);
```

`000003_add_audit_logs.down.sql`:
```sql
DROP TABLE IF EXISTS audit_logs;
```

## 迁移版本表

golang-migrate 会自动创建 `schema_migrations` 表来跟踪迁移状态：

| Column   | Type    | Description                 |
|----------|---------|-----------------------------|
| version  | INTEGER | 当前数据库版本              |
| dirty    | BOOLEAN | 是否处于脏状态（迁移失败）  |

## 注意事项

1. **迁移是不可变的** - 已应用的迁移文件不应修改
2. **始终提供 down 迁移** - 确保可以回滚
3. **测试迁移** - 在应用到生产环境前先在测试环境验证
4. **备份数据库** - 在执行重大迁移前备份数据
5. **SQLite 限制** - SQLite 对 ALTER TABLE 支持有限，某些操作需要重建表

## 故障排除

### 脏状态 (Dirty State)

如果迁移过程中断，数据库会处于脏状态：

```go
// 查看当前状态
version, dirty, _ := mm.Version()
if dirty {
    log.Printf("Database is dirty at version %d", version)

    // 如果确定问题已修复，强制设置版本
    mm.Force(version)
}
```

### 版本不匹配

如果代码期望的迁移版本高于数据库版本，会自动执行缺失的迁移。
