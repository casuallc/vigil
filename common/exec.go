package common

import (
  "bytes"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "strings"
)

// ExecuteCommand 辅助函数：执行命令
// 执行命令
func ExecuteCommand(command string, envVars []string) (string, error) {
  // 只使用bash执行命令
  cmd := exec.Command("/bin/bash", "-c", command)

  // 设置环境变量
  cmd.Env = os.Environ()
  if len(envVars) > 0 {
    cmd.Env = append(cmd.Env, envVars...)
  }

  // 捕获输出
  var stdout, stderr bytes.Buffer
  cmd.Stdout = &stdout
  cmd.Stderr = &stderr

  // 执行命令
  err := cmd.Run()

  // 合并输出
  output := stdout.String()
  if stderr.Len() > 0 {
    if output != "" {
      output += "\n"
    }
    output += stderr.String()
  }

  output = strings.TrimSpace(output)
  return output, err
}

// ExecuteScriptFile 辅助函数：执行脚本文件
func ExecuteScriptFile(filePath string, envVars []string) (string, error) {
  // 检查文件是否存在
  if _, err := os.Stat(filePath); err != nil {
    return "", fmt.Errorf("script file not found: %v", err)
  }

  var cmd *exec.Cmd

  // 根据文件扩展名确定解释器
  ext := strings.ToLower(filepath.Ext(filePath))
  switch ext {
  case ".bat", ".cmd":
    cmd = exec.Command("cmd.exe", "/c", filePath)
  case ".ps1":
    cmd = exec.Command("powershell.exe", "-File", filePath)
  case ".sh", ".bash":
    cmd = exec.Command("/bin/bash", filePath)
  case ".py":
    cmd = exec.Command("python", filePath)
  case ".go":
    // 先编译再执行Go文件
    tempDir, err := os.MkdirTemp("", "goexec")
    if err != nil {
      return "", fmt.Errorf("failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)

    tempExe := filepath.Join(tempDir, "temp.exe")
    if err := exec.Command("go", "build", "-o", tempExe, filePath).Run(); err != nil {
      return "", fmt.Errorf("failed to compile go file: %v", err)
    }
    cmd = exec.Command(tempExe)
  default:
    // 尝试直接执行（需要文件有执行权限）
    cmd = exec.Command(filePath)
  }

  // 设置环境变量
  cmd.Env = os.Environ()
  if len(envVars) > 0 {
    cmd.Env = append(cmd.Env, envVars...)
  }

  // 捕获输出
  var stdout, stderr bytes.Buffer
  cmd.Stdout = &stdout
  cmd.Stderr = &stderr

  // 执行命令
  err := cmd.Run()

  // 合并输出
  output := stdout.String()
  if stderr.Len() > 0 {
    if output != "" {
      output += "\n"
    }
    output += stderr.String()
  }

  return output, err
}
