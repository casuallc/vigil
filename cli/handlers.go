package cli

import (
  "fmt"
  "github.com/casuallc/vigil/proc"
  "gopkg.in/yaml.v3" // 导入yaml包用于YAML格式输出
  "os"
  "os/exec"
  "path/filepath"
  "time"
)

// 修改handleScan函数，显示更多进程信息并支持注册
func (c *CLI) handleScan(query string, registerAfterScan bool, namespace string) error {
  processes, err := c.client.ScanProcesses(query)
  if err != nil {
    return err
  }

  fmt.Println("Scanned processes:")
  fmt.Println("--------------------------------------------------------------------------------------------------------")
  fmt.Printf("%-5s %-8s %-20s %-15s %-25s %-15s %-20s %-40s\n",
    "#", "PID", "Name", "Status", "Start Time", "Run Time", "User", "Command")
  fmt.Println("--------------------------------------------------------------------------------------------------------")

  for i, p := range processes {
    // 计算运行时间
    var runTimeStr string
    if !p.Status.StartTime.IsZero() {
      runTime := time.Since(*p.Status.StartTime)
      runTimeStr = formatDuration(runTime)
    } else {
      runTimeStr = "N/A"
    }

    // 格式化启动时间
    startTimeStr := "N/A"
    if !p.Status.StartTime.IsZero() {
      startTimeStr = p.Status.StartTime.Format("2006-01-02 15:04:05")
    }

    // 显示进程信息，增加序号列
    fmt.Printf("%-5d %-8d %-20s %-15s %-25s %-15s %-20s %-40s\n",
      i+1, // 序号从1开始
      p.Status.PID,
      truncateString(p.Metadata.Name, 20),
      string(p.Status.Phase),
      startTimeStr,
      runTimeStr,
      truncateString(p.Spec.User, 20),
      p.Spec.Exec.Command)

    // 显示工作目录
    if p.Spec.WorkingDir != "" {
      fmt.Printf("%-5s %-8s %-20s %-15s %-25s %-15s %-20s %-40s\n",
        "", "", "", "", "", "", "Work Dir:", p.Spec.WorkingDir)
    }
  }
  fmt.Println("--------------------------------------------------------------------------------------------------------")

  // 如果需要注册进程，引导用户选择
  if registerAfterScan && len(processes) > 0 {
    return c.promptForRegistration(processes, namespace)
  }

  // 如果没有使用--register标志但有扫描结果，提示用户可以使用--register标志注册进程
  if len(processes) > 0 && !registerAfterScan {
    fmt.Println("To register a proc, use the --register flag with this command.")
  }

  return nil
}

// 修改提示用户注册进程的函数，使用与handleDeleteInteractive类似的交互式选择
func (c *CLI) promptForRegistration(processes []proc.ManagedProcess, namespace string) error {
  // 提取进程名称列表用于显示（与selectProcessInteractively相同的方式）
  processNames := make([]string, len(processes))
  for i, p := range processes {
    processNames[i] = fmt.Sprintf("%d: %s (PID: %d, Command: %s)",
      i+1, p.Metadata.Name, p.Status.PID, p.Spec.Exec.Command)
  }

  // 使用Select组件进行交互式选择
  idx, _, err := Select(SelectConfig{
    Label:    "select proc to register (use arrow keys or vim keys to navigate, press Enter to select)",
    Items:    processNames,
    PageSize: 10,
  })
  if err != nil {
    // 如果用户取消选择，不返回错误而是提示取消
    if err.Error() == "user cancelled" {
      fmt.Println("Registration cancelled.")
      return nil
    }
    return fmt.Errorf("selection failed: %v", err)
  }

  // 获取选中的进程
  selectedProcess := processes[idx]

  // 提示用户输入进程名称（可选，默认为原进程名称）
  var processName string
  fmt.Printf("Enter a name for the managed proc (default: %s): ", selectedProcess.Metadata.Name)
  fmt.Scanln(&processName)
  if processName == "" {
    processName = selectedProcess.Metadata.Name
  }

  // 创建要注册的进程
  managedProc := proc.ManagedProcess{
    Metadata: selectedProcess.Metadata,
    Spec:     selectedProcess.Spec,
    Status:   selectedProcess.Status,
  }
  managedProc.Metadata.Name = processName
  managedProc.Metadata.Namespace = namespace

  // 调用client进行注册
  err = c.client.CreateProcess(managedProc)
  if err != nil {
    return fmt.Errorf("failed to register proc: %v", err)
  }

  fmt.Printf("Successfully registered proc '%s' in namespace '%s'\n", processName, namespace)
  return nil
}

func (c *CLI) handleCreate(name, command string, namespace string) error {
  process := proc.ManagedProcess{
    Metadata: proc.Metadata{
      Name:      name,
      Namespace: namespace,
    },
    Status: proc.Status{
      Phase: proc.PhaseStopped,
    },
    Spec: proc.Spec{
      Exec: proc.Exec{
        Command: command,
      },
    },
  }

  return c.client.CreateProcess(process)
}

func (c *CLI) handleStart(name string, namespace string) error {
  return c.client.StartProcess(namespace, name)
}

// handleStartInteractive 处理交互式选择要启动的进程
func (c *CLI) handleStartInteractive(namespace string) error {
  selectedProcess, err := c.selectProcessInteractively(namespace, "select proc to start")
  if err != nil {
    fmt.Println("Err: ", err.Error())
    return nil
  }
  return c.handleStart(selectedProcess.Metadata.Name, namespace)
}

func (c *CLI) handleStop(name string, namespace string) error {
  return c.client.StopProcess(namespace, name)
}

// handleStopInteractive 处理交互式选择要停止的进程
func (c *CLI) handleStopInteractive(namespace string) error {
  selectedProcess, err := c.selectProcessInteractively(namespace, "select proc to stop")
  if err != nil {
    fmt.Println("Err: ", err.Error())
    return nil
  }
  return c.handleStop(selectedProcess.Metadata.Name, namespace)
}

func (c *CLI) handleRestart(name string, namespace string) error {
  return c.client.RestartProcess(namespace, name)
}

// handleRestartInteractive 处理交互式选择要重启的进程
func (c *CLI) handleRestartInteractive(namespace string) error {
  selectedProcess, err := c.selectProcessInteractively(namespace, "select proc to restart")
  if err != nil {
    fmt.Println("Err: ", err.Error())
    return nil
  }
  return c.handleRestart(selectedProcess.Metadata.Name, namespace)
}

func (c *CLI) handleList(namespace string) error {
  processes, err := c.client.ListProcesses(namespace)
  if err != nil {
    return err
  }

  fmt.Println("Managed processes:")
  for _, p := range processes {
    fmt.Printf("Name: %s, Status: %s, PID: %d, Namespace: %s\n", p.Metadata.Name, p.Status.Phase, p.Status.PID, p.Metadata.Namespace)
  }

  return nil
}

func (c *CLI) handleStatus(name string, namespace string) error {
  process, err := c.client.GetProcess(namespace, name)
  if err != nil {
    return err
  }

  fmt.Printf("Process status for '%s' in namespace '%s':\n", name, namespace)
  fmt.Printf("  Status: %s\n", process.Status.Phase)
  fmt.Printf("  PID: %d\n", process.Status.PID)
  fmt.Printf("  Command: %s\n", process.Spec.Exec.Command)
  fmt.Printf("  Working Directory: %s\n", process.Spec.WorkingDir)
  fmt.Printf("  Start Time: %s\n", process.Status.StartTime.Format("2006-01-02 15:04:05"))
  fmt.Printf("  Restart Count: %d\n", process.Status.RestartCount)
  fmt.Printf("  Last Exit Code: %d\n", process.Status.LastTerminationInfo.ExitCode)

  return nil
}

// handleGet 处理get命令，支持输出YAML格式
func (c *CLI) handleGet(name string, format string, namespace string) error {
  process, err := c.client.GetProcess(namespace, name)
  if err != nil {
    return err
  }

  if format == "yaml" {
    // 以YAML格式输出进程信息
    yamlData, err := yaml.Marshal(process)
    if err != nil {
      return fmt.Errorf("failed to marshal proc data to YAML: %v", err)
    }
    fmt.Println(string(yamlData))
  } else {
    // 默认以文本格式输出，与status命令类似但增加更多信息
    fmt.Printf("Process details for '%s' in namespace '%s':\n", name, namespace)
    fmt.Printf("  Status: %s\n", process.Status.Phase)
    fmt.Printf("  PID: %d\n", process.Status.PID)
    fmt.Printf("  Command: %s\n", process.Spec.Exec.Command)
    if len(process.Spec.Exec.Args) > 0 {
      fmt.Printf("  Args: %v\n", process.Spec.Exec.Args)
    }
    fmt.Printf("  Working Directory: %s\n", process.Spec.WorkingDir)
    fmt.Printf("  Start Time: %s\n", process.Status.StartTime.Format("2006-01-02 15:04:05"))
    fmt.Printf("  Restart Policy: %s\n", process.Spec.RestartPolicy)
    fmt.Printf("  Restart Count: %d\n", process.Status.RestartCount)
    fmt.Printf("  Restart Interval: %s\n", process.Spec.RestartInterval)
    fmt.Printf("  Last Exit Code: %d\n", process.Status.LastTerminationInfo.ExitCode)
    if process.Spec.User != "" {
      fmt.Printf("  User: %s\n", process.Spec.User)
    }
    if process.Spec.UserGroup != "" {
      fmt.Printf("  User Group: %s\n", process.Spec.UserGroup)
    }
    if process.Spec.Log.Dir != "" {
      fmt.Printf("  Log Directory: %s\n", process.Spec.Log.Dir)
    }

    // 如果有环境变量，输出它们
    if len(process.Spec.Env) > 0 {
      fmt.Println("  Environment Variables:")
      for _, env := range process.Spec.Env {
        fmt.Printf("    %s=%s\n", env.Name, env.Value)
      }
    }
  }

  return nil
}

// handleGetInteractive 处理交互式选择进程的情况
func (c *CLI) handleGetInteractive(format string, namespace string) error {
  selectedProcess, err := c.selectProcessInteractively(namespace, "select proc")
  if err != nil {
    fmt.Println("Err: ", err.Error())
    return nil
  }
  return c.handleGet(selectedProcess.Metadata.Name, format, namespace)
}

// handleEdit 处理编辑进程定义的命令
func (c *CLI) handleEdit(name string, namespace string) error {
  // 1. 获取进程信息
  process, err := c.client.GetProcess(namespace, name)
  if err != nil {
    return fmt.Errorf("failed to get process: %v", err)
  }

  // 2. 创建临时文件
  tmpDir, err := os.MkdirTemp("", "vigil-edit-")
  if err != nil {
    return fmt.Errorf("failed to create temporary directory: %v", err)
  }
  defer func(path string) {
    err := os.RemoveAll(path)
    if err != nil {
      fmt.Printf("failed to remove temporary directory: %v", err)
    }
  }(tmpDir) // 确保清理临时文件

  tmpFile := filepath.Join(tmpDir, fmt.Sprintf("%s-%s.yaml", namespace, name))

  // 3. 将进程信息序列化为YAML并写入临时文件
  yamlData, err := yaml.Marshal(process)
  if err != nil {
    return fmt.Errorf("failed to marshal process data: %v", err)
  }

  if err := os.WriteFile(tmpFile, yamlData, 0644); err != nil {
    return fmt.Errorf("failed to write temporary file: %v", err)
  }

  // 4. 打开vim编辑器
  editor := os.Getenv("EDITOR")
  if editor == "" {
    editor = "vim" // 默认使用vim
  }

  cmd := exec.Command(editor, tmpFile)
  cmd.Stdin = os.Stdin
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr

  if err := cmd.Run(); err != nil {
    return fmt.Errorf("editor exited with error: %v", err)
  }

  // 5. 读取编辑后的文件内容
  editedData, err := os.ReadFile(tmpFile)
  if err != nil {
    return fmt.Errorf("failed to read edited file: %v", err)
  }

  // 6. 解析编辑后的YAML数据
  var updatedProc proc.ManagedProcess
  if err := yaml.Unmarshal(editedData, &updatedProc); err != nil {
    return fmt.Errorf("failed to parse edited data: %v", err)
  }

  // 7. 确保名称和命名空间不变
  updatedProc.Metadata.Name = name
  updatedProc.Metadata.Namespace = namespace

  // 8. 发送更新请求
  if err := c.client.UpdateProcess(updatedProc); err != nil {
    return fmt.Errorf("failed to update process: %v", err)
  }

  fmt.Printf("Successfully updated process '%s' in namespace '%s'\n", name, namespace)
  return nil
}

// handleEditInteractive 处理交互式编辑进程的命令
func (c *CLI) handleEditInteractive(namespace string) error {
  selectedProcess, err := c.selectProcessInteractively(namespace, "select proc to edit")
  if err != nil {
    fmt.Println("Err: ", err.Error())
    return nil
  }
  return c.handleEdit(selectedProcess.Metadata.Name, namespace)
}

// handleDelete handles the delete command to remove a managed proc
func (c *CLI) handleDelete(name string, namespace string) error {
  err := c.client.DeleteProcess(namespace, name)
  if err != nil {
    return fmt.Errorf("删除进程失败: %w", err)
  }

  fmt.Printf("进程 %s (命名空间: %s) 删除成功\n", name, namespace)
  return nil
}

// handleDeleteInteractive 处理交互式选择要删除的进程
func (c *CLI) handleDeleteInteractive(namespace string) error {
  selectedProcess, err := c.selectProcessInteractively(namespace, "select proc to delete")
  if err != nil {
    fmt.Println("Err: ", err.Error())
    return nil
  }
  return c.handleDelete(selectedProcess.Metadata.Name, namespace)
}

func (c *CLI) handleSystemResources() error {
  resources, err := c.client.GetSystemResources()
  if err != nil {
    return err
  }

  fmt.Println("System Resource Usage:")
  fmt.Printf("  CPU Usage: %.2f%%\n", resources.CPUUsage)
  fmt.Printf("  Memory Usage: %d bytes\n", resources.MemoryUsage)
  fmt.Printf("  Disk IO: %d bytes\n", resources.DiskIO)
  fmt.Printf("  Network IO: %d bytes\n", resources.NetworkIO)

  return nil
}

func (c *CLI) handleProcessResources(pid int) error {
  resources, err := c.client.GetProcessResources(pid)
  if err != nil {
    return err
  }

  fmt.Printf("Resource Usage for Process %d:\n", pid)
  fmt.Printf("  CPU Usage: %.2f%%\n", resources.CPUUsage)
  fmt.Printf("  Memory Usage: %d bytes\n", resources.MemoryUsage)
  fmt.Printf("  Disk IO: %d bytes\n", resources.DiskIO)
  fmt.Printf("  Network IO: %d bytes\n", resources.NetworkIO)

  return nil
}

func (c *CLI) handleGetConfig() error {
  cfg, err := c.client.GetConfig()
  if err != nil {
    return err
  }

  fmt.Println("Current Configuration:")
  fmt.Printf("  Log Level: %s\n", cfg.LogLevel)
  fmt.Printf("  Monitor Rate: %d seconds\n", cfg.MonitorRate)
  fmt.Printf("  PID File Path: %s\n", cfg.PidFilePath)
  fmt.Printf("  Managed Apps Count: %d\n", len(cfg.ManagedApps))

  return nil
}

// handleExec handles the exec command to execute a command or script
func (c *CLI) handleExec(command string, isFile bool, envVars []string, outputFile string) error {
  // 如果是文件，读取本地文件内容
  if isFile {
    fileContent, err := os.ReadFile(command)
    if err != nil {
      return fmt.Errorf("failed to read script file: %w", err)
    }
    // 将文件内容作为命令发送，标记为非文件
    command = string(fileContent)
    isFile = false
  }

  // 执行命令并获取输出
  output, err := c.client.ExecuteCommand(command, isFile, envVars)

  // 根据outputFile参数决定输出目标
  if outputFile != "" {
    // 输出到文件
    if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
      return fmt.Errorf("failed to write output to file: %w", err)
    }
    fmt.Printf("Command output written to %s\n", outputFile)
  } else {
    // 输出到控制台
    fmt.Println(output)
  }

  // 如果有错误，返回错误信息
  if err != nil {
    return fmt.Errorf("command execution failed: %w", err)
  }

  return nil
}

// 新增：添加挂载
func (c *CLI) handleProcMountAdd(name, namespace string, mountID string, mountType string, target, source, volumeName string, readOnly bool, options []string) error {
  p, err := c.client.GetProcess(namespace, name)
  if err != nil {
    return fmt.Errorf("failed to get process: %w", err)
  }

  m := proc.Mount{
    ID:       mountID,
    Type:     mountType,
    Target:   target,
    ReadOnly: readOnly,
    Options:  options,
  }

  switch mountType {
  case "bind":
    m.Source = source
  case "volume":
    m.Name = volumeName
  case "tmpfs":
    // no extra fields
  default:
    return fmt.Errorf("unsupported mount type: %s", mountType)
  }

  // 若已存在同名(ID)挂载，则直接忽略（按名称判断重复）
  for _, existing := range p.Spec.Mounts {
    if existing.ID != "" && existing.ID == m.ID {
      fmt.Printf("Mount '%s' already exists for process '%s' (ns=%s), no changes applied.\n", mountID, name, namespace)
      return nil
    }
  }

  p.Spec.Mounts = append(p.Spec.Mounts, m)

  if err := c.client.UpdateProcess(p); err != nil {
    return fmt.Errorf("failed to update process mounts: %w", err)
  }

  fmt.Printf("Added mount '%s' to process '%s' (ns=%s): type=%s target=%s\n", mountID, name, namespace, mountType, target)
  return nil
}

// 新增：移除挂载（支持按target或按index）
func (c *CLI) handleProcMountRemove(name, namespace, target string, index int) error {
  p, err := c.client.GetProcess(namespace, name)
  if err != nil {
    return fmt.Errorf("failed to get process: %w", err)
  }

  mounts := p.Spec.Mounts
  if len(mounts) == 0 {
    fmt.Println("No mounts configured.")
    return nil
  }

  var newMounts []proc.Mount

  if target != "" {
    for i := range mounts {
      if mounts[i].Target != target {
        newMounts = append(newMounts, mounts[i])
      }
    }
    if len(newMounts) == len(mounts) {
      return fmt.Errorf("no mount found with target: %s", target)
    }
  } else if index >= 0 {
    if index < 0 || index >= len(mounts) {
      return fmt.Errorf("index out of range: %d", index)
    }
    newMounts = append(newMounts, mounts[:index]...)
    newMounts = append(newMounts, mounts[index+1:]...)
  } else {
    return fmt.Errorf("either target or index must be specified")
  }

  p.Spec.Mounts = newMounts
  if err := c.client.UpdateProcess(p); err != nil {
    return fmt.Errorf("failed to update process mounts: %w", err)
  }

  fmt.Printf("Removed mount from process '%s' (ns=%s)\n", name, namespace)
  return nil
}

// 新增：列出挂载
func (c *CLI) handleProcMountList(name, namespace string) error {
  p, err := c.client.GetProcess(namespace, name)
  if err != nil {
    return fmt.Errorf("failed to get process: %w", err)
  }

  mounts := p.Spec.Mounts
  if len(mounts) == 0 {
    fmt.Println("No mounts configured.")
    return nil
  }

  fmt.Printf("Mounts for process '%s' (ns=%s):\n", name, namespace)
  fmt.Println("------------------------------------------------------------------------------------------")
  fmt.Printf("%-5s %-8s %-25s %-35s %-8s %-20s\n", "#", "Type", "Source/Name", "Target", "RO", "Options")
  fmt.Println("------------------------------------------------------------------------------------------")
  for i, m := range mounts {
    src := m.Source
    if m.Type == "volume" {
      src = m.Name
    }
    fmt.Printf("%-5d %-8s %-25s %-35s %-8t %-20v\n", i, string(m.Type), src, m.Target, m.ReadOnly, m.Options)
  }
  return nil
}
