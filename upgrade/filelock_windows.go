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

//go:build windows

package upgrade

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFile acquires a file lock (shared or exclusive) on Windows
func lockFile(f *os.File, exclusive bool) error {
	var flags uint32
	if exclusive {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	// Lock entire file
	var ol windows.Overlapped
	h := windows.Handle(f.Fd())
	return windows.LockFileEx(h, flags, 0, 0xFFFFFFFF, 0xFFFFFFFF, &ol)
}

// unlockFile releases the file lock on Windows
func unlockFile(f *os.File) error {
	var ol windows.Overlapped
	h := windows.Handle(f.Fd())
	return windows.UnlockFileEx(h, 0, 0xFFFFFFFF, 0xFFFFFFFF, &ol)
}

// processExists checks if a process exists on Windows
func processExists(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	var code uint32
	err = windows.GetExitCodeProcess(handle, &code)
	return err == nil && code == 259 // STILL_ACTIVE
}
