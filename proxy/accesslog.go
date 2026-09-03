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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileAccessHook returns an AccessHook that appends one JSON line per
// record to logs/proxy/<instance>_YYYY-MM-DD.log under dir.
func FileAccessHook(dir string) AccessHook {
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Fall back to a no-op hook rather than failing startup.
		return func(AccessRecord) {}
	}
	files := &accessLogFiles{dir: dir, files: make(map[string]*os.File)}
	return files.record
}

// accessLogFiles keeps one open file per (instance, day) pair.
type accessLogFiles struct {
	dir   string
	mu    sync.Mutex
	files map[string]*os.File
	day   string
}

func (f *accessLogFiles) record(rec AccessRecord) {
	line, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	day := time.Now().Format("2006-01-02")
	if day != f.day {
		// Day rolled over: reopen all files lazily.
		for _, file := range f.files {
			file.Close()
		}
		f.files = make(map[string]*os.File)
		f.day = day
	}
	name := rec.Instance
	if name == "" {
		name = "tunnel"
	}
	file, ok := f.files[name]
	if !ok {
		path := filepath.Join(f.dir, fmt.Sprintf("%s_%s.log", name, day))
		file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		f.files[name] = file
	}
	file.Write(append(line, '\n'))
}
