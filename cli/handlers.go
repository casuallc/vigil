package cli

import (
  "fmt"
  "github.com/casuallc/vigil/process"
  "gopkg.in/yaml.v2" // 导入yaml包用于YAML格式输出
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
    fmt.Println("To register a process, use the --register flag with this command.")
  }

  return nil
}

// 添加提示用户注册进程的函数
func (c *CLI) promptForRegistration(processes []process.ManagedProcess, namespace string) error {
  var choice int
  fmt.Print("Enter the number of the process you want to register (0 to cancel): ")
  _, err := fmt.Scanf("%d", &choice)
  if err != nil {
    return fmt.Errorf("invalid input: %v", err)
  }

  if choice == 0 {
    fmt.Println("Registration cancelled.")
    return nil
  }

  if choice < 1 || choice > len(processes) {
    return fmt.Errorf("invalid choice: %d. Please enter a number between 1 and %d", choice, len(processes))
  }

  // 获取用户选择的进程
  selectedProcess := processes[choice-1]

  // 提示用户输入进程名称（可选，默认为原进程名称）
  var processName string
  fmt.Printf("Enter a name for the managed process (default: %s): ", selectedProcess.Metadata.Name)
  fmt.Scanln(&processName)
  if processName == "" {
    processName = selectedProcess.Metadata.Name
  }

  // 创建要注册的进程
  managedProcess := process.ManagedProcess{
    Metadata: selectedProcess.Metadata,
    Spec:     selectedProcess.Spec,
    Status:   selectedProcess.Status,
  }
  managedProcess.Metadata.Name = processName
  managedProcess.Metadata.Namespace = namespace

  // 调用client进行注册
  err = c.client.ManageProcess(managedProcess)
  if err != nil {
    return fmt.Errorf("failed to register process: %v", err)
  }

  fmt.Printf("Successfully registered process '%s' in namespace '%s'\n", processName, namespace)
  return nil
}

func (c *CLI) handleManage(name, command string, namespace string) error {
  process := process.ManagedProcess{
    Metadata: process.Metadata{
      Name:      name,
      Namespace: namespace,
    },
    Status: process.Status{
      Phase: process.PhaseStopped,
    },
    Spec: process.Spec{
      Exec: process.Exec{
        Command: command,
      },
    },
  }

  return c.client.ManageProcess(process)
}

func (c *CLI) handleStart(name string, namespace string) error {
  return c.client.StartProcess(namespace, name)
}

func (c *CLI) handleStop(name string, namespace string) error {
  return c.client.StopProcess(namespace, name)
}

func (c *CLI) handleRestart(name string, namespace string) error {
  return c.client.RestartProcess(namespace, name)
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
      return fmt.Errorf("failed to marshal process data to YAML: %v", err)
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
    fmt.Printf("  Max Restarts: %d\n", process.Spec.MaxRestarts)
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

// handleDelete handles the delete command to remove a managed process
func (c *CLI) handleDelete(name string, namespace string) error {
  err := c.client.DeleteProcess(namespace, name)
  if err != nil {
    return fmt.Errorf("删除进程失败: %w", err)
  }

  fmt.Printf("进程 %s (命名空间: %s) 删除成功\n", name, namespace)
  return nil
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
