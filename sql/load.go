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
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var schemaFiles embed.FS

// LoadSchema 加载指定 SQL 文件的 schema 内容
func LoadSchema(filename string) (string, error) {
	content, err := schemaFiles.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file %s: %w", filename, err)
	}
	return string(content), nil
}

// LoadUsersSchema 加载用户表 schema
func LoadUsersSchema() (string, error) {
	return LoadSchema("users.sql")
}

// LoadVMsSchema 加载 VM 表 schema
func LoadVMsSchema() (string, error) {
	return LoadSchema("vms.sql")
}

// LoadProcsSchema 加载进程表 schema
func LoadProcsSchema() (string, error) {
	return LoadSchema("procs.sql")
}

// SplitStatements 将 SQL 内容分割成单独的语句
func SplitStatements(sql string) []string {
	// 按分号分割 SQL 语句
	statements := strings.Split(sql, ";")
	var result []string
	for _, stmt := range statements {
		// 去除空白
		stmt = strings.TrimSpace(stmt)
		// 跳过空语句和注释
		if stmt != "" && !strings.HasPrefix(stmt, "--") {
			result = append(result, stmt)
		}
	}
	return result
}
