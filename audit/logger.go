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

package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Logger 审计日志记录器
type Logger struct {
	logDir   string
	logFile  string
	mu       sync.Mutex
	logFileHandle *os.File
}

// NewLogger 创建新的审计日志记录器
func NewLogger(logDir string) (*Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// 创建日志文件名
	logFile := filepath.Join(logDir, fmt.Sprintf("audit_%s.log", time.Now().Format("2006-01-02")))

	// 打开或创建日志文件
	logFileHandle, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	return &Logger{
		logDir:   logDir,
		logFile:  logFile,
		logFileHandle: logFileHandle,
	}, nil
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFileHandle != nil {
		return l.logFileHandle.Close()
	}

	return nil
}

// Log 记录审计日志
func (l *Logger) Log(entry *LogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查日志文件是否需要轮换
	currentLogFile := filepath.Join(l.logDir, fmt.Sprintf("audit_%s.log", time.Now().Format("2006-01-02")))
	if currentLogFile != l.logFile {
		// 关闭当前日志文件
		if l.logFileHandle != nil {
			l.logFileHandle.Close()
		}

		// 打开新的日志文件
		logFileHandle, err := os.OpenFile(currentLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open new log file: %v", err)
		}

		l.logFile = currentLogFile
		l.logFileHandle = logFileHandle
	}

	// 将日志条目转换为JSON
	logJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %v", err)
	}

	// 写入日志文件
	if _, err := l.logFileHandle.WriteString(string(logJSON) + "\n"); err != nil {
		return fmt.Errorf("failed to write log entry: %v", err)
	}

	// 刷新缓冲区
	if err := l.logFileHandle.Sync(); err != nil {
		return fmt.Errorf("failed to sync log file: %v", err)
	}

	return nil
}

// generateID 生成唯一ID
func generateID() string {
	return uuid.New().String()
}
