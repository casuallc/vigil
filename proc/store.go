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

package proc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/casuallc/vigil/models"
	dbsql "github.com/casuallc/vigil/sql"
	_ "modernc.org/sqlite"
)

// ProcessStore 进程存储管理器
type ProcessStore struct {
	db *sql.DB
}

// NewProcessStore 创建新的进程存储管理器
func NewProcessStore(dbPath string) (*ProcessStore, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 打开数据库
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &ProcessStore{db: db}

	// 初始化 schema
	if err := store.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initDB 初始化数据库 schema（使用 migrations）
func (s *ProcessStore) initDB() error {
	// Run database migrations (procs table is created by migrations)
	if err := dbsql.InitAndMigrate(s.db); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}
	return nil
}

// Close 关闭数据库连接
func (s *ProcessStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveManagedProcesses 保存所有已管理的进程到数据库
func (s *ProcessStore) SaveManagedProcesses(processes map[string]*models.ManagedProcess) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO procs
		(namespace, name, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for key, p := range processes {
		// 过滤掉运行时的状态信息，只保存配置相关信息
		processCopy := *p
		processCopy.Status.Phase = models.PhaseFailed
		processCopy.Status.PID = 0
		processCopy.Status.StartTime = &time.Time{}
		processCopy.Status.ResourceStats = nil

		configJSON, err := json.Marshal(processCopy)
		if err != nil {
			return fmt.Errorf("failed to marshal process %s: %w", key, err)
		}

		_, err = stmt.Exec(
			p.Metadata.Namespace,
			p.Metadata.Name,
			configJSON,
			p.Metadata.CreationTimestamp,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to save process %s: %w", key, err)
		}
	}

	return tx.Commit()
}

// LoadManagedProcesses 从数据库加载已管理的进程
func (s *ProcessStore) LoadManagedProcesses(m *Manager) error {
	rows, err := s.db.Query(`SELECT namespace, name, config FROM procs`)
	if err != nil {
		return fmt.Errorf("failed to query processes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var namespace, name string
		var configJSON []byte

		err := rows.Scan(&namespace, &name, &configJSON)
		if err != nil {
			return fmt.Errorf("failed to scan process row: %w", err)
		}

		var process models.ManagedProcess
		if err := json.Unmarshal(configJSON, &process); err != nil {
			return fmt.Errorf("failed to unmarshal process config: %w", err)
		}

		// 使用 namespace/name 作为键
		key := fmt.Sprintf("%s/%s", namespace, name)
		m.Processes[key] = &process

		// 启动监控协程（避免重复）
		m.StartMonitoring(namespace, name)

		// 自动启动标记为需要重启的进程
		if process.Spec.RestartPolicy == models.RestartPolicyAlways ||
			(process.Spec.RestartPolicy == models.RestartPolicyOnFailure && process.Status.LastTerminationInfo.ExitCode != 0) {
			go func(namespace, name string) {
				// 延迟启动，避免启动时资源竞争
				time.Sleep(1 * time.Second)
				if err := m.StartProcess(namespace, name); err != nil {
					log.Printf("Failed to start proc %s/%s on startup: %v\n", namespace, name, err)
				}
			}(namespace, name)
		}
	}

	return rows.Err()
}
