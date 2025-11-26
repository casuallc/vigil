package version

import (
  "fmt"
  "runtime"
)

// 版本信息变量，将在构建时通过 ldflags 注入
var (
  Version   = "1.0"
  BuildTime = "unknown"
  GitCommit = "unknown"
  GitBranch = "unknown"
)

// Info 包含完整的版本信息
type Info struct {
  Version   string
  BuildTime string
  GitCommit string
  GitBranch string
  GoVersion string
  OS        string
  Arch      string
}

// GetVersionInfo 返回完整的版本信息
func GetVersionInfo() Info {
  return Info{
    Version:   Version,
    BuildTime: BuildTime,
    GitCommit: GitCommit,
    GitBranch: GitBranch,
    GoVersion: runtime.Version(),
    OS:        runtime.GOOS,
    Arch:      runtime.GOARCH,
  }
}

// String 返回格式化的版本信息字符串
func (v Info) String() string {
  return fmt.Sprintf(`Version:   %s
BuildTime: %s
GitCommit: %s
GitBranch: %s
GoVersion: %s
OS/Arch:   %s/%s`,
    v.Version,
    v.BuildTime,
    v.GitCommit,
    v.GitBranch,
    v.GoVersion,
    v.OS,
    v.Arch,
  )
}
