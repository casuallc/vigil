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

package proxy

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dbsql "github.com/casuallc/vigil/sql"
	_ "modernc.org/sqlite"
)

// Desired states persisted for API-created instances.
const (
	desiredRunning = "running"
	desiredStopped = "stopped"
)

// storedInstance is one row of the proxy_instances table.
type storedInstance struct {
	cfg     InstanceConfig
	origin  string
	desired string
}

// store persists API-managed proxy instances in the shared SQLite database.
type store struct {
	db *sql.DB
}

func newStore(dbPath string) (*store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	schema, err := dbsql.LoadProxyInstancesSchema()
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize proxy_instances schema: %w", err)
	}
	return &store{db: db}, nil
}

func (s *store) close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// upsert inserts or replaces an instance record.
func (s *store) upsert(cfg InstanceConfig, origin, desired string) error {
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal instance config: %w", err)
	}
	_, err = s.db.Exec(`INSERT OR REPLACE INTO proxy_instances
		(name, origin, config_json, desired_state, created_at, updated_at)
		VALUES (?, ?, ?, ?,
			COALESCE((SELECT created_at FROM proxy_instances WHERE name = ?), ?), ?)`,
		cfg.Name, origin, string(configJSON), desired, cfg.Name, time.Now(), time.Now())
	return err
}

func (s *store) delete(name string) error {
	_, err := s.db.Exec(`DELETE FROM proxy_instances WHERE name = ?`, name)
	return err
}

func (s *store) setDesired(name, desired string) error {
	res, err := s.db.Exec(`UPDATE proxy_instances SET desired_state = ?, updated_at = ? WHERE name = ?`,
		desired, time.Now(), name)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return fmt.Errorf("proxy: instance %q not found", name)
	}
	return nil
}

// loadAll returns every stored instance.
func (s *store) loadAll() ([]storedInstance, error) {
	rows, err := s.db.Query(`SELECT name, origin, config_json, desired_state FROM proxy_instances`)
	if err != nil {
		return nil, fmt.Errorf("failed to query proxy instances: %w", err)
	}
	defer rows.Close()

	var out []storedInstance
	for rows.Next() {
		var name, origin, configJSON, desired string
		if err := rows.Scan(&name, &origin, &configJSON, &desired); err != nil {
			return nil, fmt.Errorf("failed to scan proxy instance row: %w", err)
		}
		var cfg InstanceConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config of instance %q: %w", name, err)
		}
		cfg.Name = name
		out = append(out, storedInstance{cfg: cfg, origin: origin, desired: desired})
	}
	return out, rows.Err()
}
