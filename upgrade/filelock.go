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

//go:build !windows

package upgrade

import (
	"os"
	"syscall"
)

// lockFile acquires a file lock (shared or exclusive)
func lockFile(f *os.File, exclusive bool) error {
	var how int
	if exclusive {
		how = syscall.LOCK_EX
	} else {
		how = syscall.LOCK_SH
	}
	return syscall.Flock(int(f.Fd()), how|syscall.LOCK_NB)
}

// unlockFile releases the file lock
func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

// processExists checks if a process exists by sending signal 0
func processExists(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
