//go:build !windows

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

package file

import (
  "fmt"
  "os"
  "os/user"
  "strconv"
  "strings"
)

// chownPath 设置文件所属用户，owner 格式为 "user" 或 "user:group"
func chownPath(path, owner string) error {
  parts := strings.SplitN(owner, ":", 2)

  uid, err := lookupUID(parts[0])
  if err != nil {
    return err
  }

  gid := -1 // 保持不变
  if len(parts) == 2 && parts[1] != "" {
    gid, err = lookupGID(parts[1])
    if err != nil {
      return err
    }
  }

  if err := os.Chown(path, uid, gid); err != nil {
    return fmt.Errorf("failed to chown to %s: %v", owner, err)
  }
  return nil
}

// lookupUID 支持用户名或数字 UID
func lookupUID(nameOrID string) (int, error) {
  if id, err := strconv.Atoi(nameOrID); err == nil {
    return id, nil
  }
  u, err := user.Lookup(nameOrID)
  if err != nil {
    return 0, fmt.Errorf("unknown user %q: %v", nameOrID, err)
  }
  return strconv.Atoi(u.Uid)
}

// lookupGID 支持组名或数字 GID
func lookupGID(nameOrID string) (int, error) {
  if id, err := strconv.Atoi(nameOrID); err == nil {
    return id, nil
  }
  g, err := user.LookupGroup(nameOrID)
  if err != nil {
    return 0, fmt.Errorf("unknown group %q: %v", nameOrID, err)
  }
  return strconv.Atoi(g.Gid)
}
