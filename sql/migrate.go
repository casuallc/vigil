/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sql

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// MigrationManager 数据库迁移管理器
type MigrationManager struct {
	migrate *migrate.Migrate
}

// NewMigrationManager 创建新的迁移管理器
// dbPath: SQLite 数据库文件路径 (例如: "file:/path/to/db.sqlite?cache=shared")
func NewMigrationManager(db *sql.DB) (*MigrationManager, error) {
	// 创建 sqlite 数据库驱动实例
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	// 从 embed.FS 加载迁移文件
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// 创建 migrate 实例
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return &MigrationManager{
		migrate: m,
	}, nil
}

// Up 执行所有待执行的迁移
func (mm *MigrationManager) Up() error {
	if err := mm.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}
	return nil
}

// UpTo 迁移到指定版本
func (mm *MigrationManager) UpTo(version uint) error {
	if err := mm.migrate.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}
	return nil
}

// Down 回滚一个版本
func (mm *MigrationManager) Down() error {
	if err := mm.migrate.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback one version: %w", err)
	}
	return nil
}

// DownTo 回滚到指定版本
func (mm *MigrationManager) DownTo(version uint) error {
	if err := mm.migrate.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate down to version %d: %w", version, err)
	}
	return nil
}

// Drop 删除所有表（回滚到初始状态）
func (mm *MigrationManager) Drop() error {
	if err := mm.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to drop all tables: %w", err)
	}
	return nil
}

// Version 获取当前数据库版本
// 返回值: (version, isDirty, error)
// version: 当前版本号
// isDirty: 如果上次迁移失败则为 true
func (mm *MigrationManager) Version() (uint, bool, error) {
	return mm.migrate.Version()
}

// Force 强制设置数据库版本（用于修复脏状态）
func (mm *MigrationManager) Force(version uint) error {
	if err := mm.migrate.Force(int(version)); err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}
	return nil
}

// Close 关闭迁移管理器
func (mm *MigrationManager) Close() error {
	sourceErr, dbErr := mm.migrate.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close migration source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close migration db: %w", dbErr)
	}
	return nil
}

// CloseSourceOnly 仅关闭迁移源，不关闭数据库连接
// 用于外部管理数据库连接生命周期的场景
func (mm *MigrationManager) CloseSourceOnly() error {
	sourceErr, _ := mm.migrate.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close migration source: %w", sourceErr)
	}
	return nil
}

// InitAndMigrate 初始化并执行迁移（便捷方法）
// 这个函数会创建迁移表（如果不存在）并执行所有待执行的迁移
// 注意：此函数不会关闭传入的数据库连接，也不会关闭内部资源（让 GC 处理）
func InitAndMigrate(db *sql.DB) error {
	mm, err := NewMigrationManager(db)
	if err != nil {
		return err
	}
	// 注意：我们不调用 mm.Close()，因为 Close 会关闭数据库连接
	// 而 db 是外部传入的，生命周期由外部管理
	// golang-migrate 的内部资源会在 GC 时自动清理

	// 获取当前版本
	version, dirty, err := mm.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	// 如果数据库处于脏状态，可能需要强制设置版本
	if dirty {
		log.Printf("Warning: Database is in dirty state at version %d, attempting to fix...", version)
		if err := mm.Force(version); err != nil {
			return err
		}
	}

	// 执行迁移
	if err := mm.Up(); err != nil {
		return err
	}

	// 记录迁移后的版本
	newVersion, _, err := mm.Version()
	if err != nil {
		return fmt.Errorf("failed to get migration version after up: %w", err)
	}

	if version != newVersion {
		log.Printf("Database migrated from version %d to %d", version, newVersion)
	} else {
		log.Printf("Database is up to date at version %d", version)
	}

	return nil
}

// InitAndMigrateWithPath 从数据库路径初始化并执行迁移（最常用方法）
func InitAndMigrateWithPath(dbPath string) error {
	// 打开数据库连接
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	return InitAndMigrate(db)
}
