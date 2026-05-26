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
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func replaceBinary(src, dst string) error {
	// On Windows, old process must exit first, then we can replace
	// Try to remove old file first
	if err := os.Remove(dst); err != nil {
		if !os.IsNotExist(err) {
			// File still locked by old process, schedule rename on reboot
			return scheduleReplaceOnReboot(src, dst)
		}
	}
	return os.Rename(src, dst)
}

func scheduleReplaceOnReboot(src, dst string) error {
	srcPtr, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dstPtr, err := windows.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}

	err = windows.MoveFileEx(srcPtr, dstPtr, windows.MOVEFILE_DELAY_UNTIL_REBOOT)
	if err != nil {
		return fmt.Errorf("MoveFileEx failed: %w", err)
	}
	return nil
}
