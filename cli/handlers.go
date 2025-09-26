package cli

import (
  "fmt"
  "github.com/casuallc/vigil/process"
  "time"
)

// 修改handleScan函数，显示更多进程信息并支持注册
func (c *CLI) handleScan(query string, registerAfterScan bool) error {
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
    if !p.StartTime.IsZero() {
      runTime := time.Since(p.StartTime)
      runTimeStr = formatDuration(runTime)
    } else {
      runTimeStr = "N/A"
    }

    // 格式化启动时间
    startTimeStr := "N/A"
    if !p.StartTime.IsZero() {
      startTimeStr = p.StartTime.Format("2006-01-02 15:04:05")
    }

    // 显示进程信息，增加序号列
    fmt.Printf("%-5d %-8d %-20s %-15s %-25s %-15s %-20s %-40s\n",
      i+1, // 序号从1开始
      p.PID,
      truncateString(p.Name, 20),
      string(p.Status),
      startTimeStr,
      runTimeStr,
      truncateString(p.User, 20),
      p.StartCommand.Command)

    // 显示工作目录
    if p.WorkingDir != "" {
      fmt.Printf("%-5s %-8s %-20s %-15s %-25s %-15s %-20s %-40s\n",
        "", "", "", "", "", "", "Work Dir:", p.WorkingDir)
    }
  }
  fmt.Println("--------------------------------------------------------------------------------------------------------")

  // 如果需要注册进程，引导用户选择
  if registerAfterScan && len(processes) > 0 {
    return c.promptForRegistration(processes)
  }

  // 如果没有使用--register标志但有扫描结果，提示用户可以使用--register标志注册进程
  if len(processes) > 0 && !registerAfterScan {
    fmt.Println("To register a process, use the --register flag with this command.")
  }

  return nil
}

// 添加提示用户注册进程的函数
func (c *CLI) promptForRegistration(processes []process.ManagedProcess) error {
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
  fmt.Printf("Enter a name for the managed process (default: %s): ", selectedProcess.Name)
  fmt.Scanln(&processName)
  if processName == "" {
    processName = selectedProcess.Name
  }

  // 创建要注册的进程
  managedProcess := process.ManagedProcess{
    Name:         processName,
    Status:       process.StatusRunning, // 因为进程已经在运行
    PID:          selectedProcess.PID,
    StartCommand: selectedProcess.StartCommand,
    WorkingDir:   selectedProcess.WorkingDir,
    User:         selectedProcess.User,
    UserGroup:    selectedProcess.UserGroup,
  }

  // 调用client进行注册
  err = c.client.ManageProcess(managedProcess)
  if err != nil {
    return fmt.Errorf("failed to register process: %v", err)
  }

  fmt.Printf("Successfully registered process '%s'\n", processName)
  return nil
}

func (c *CLI) handleManage(name, command string) error {
  process := process.ManagedProcess{
    Name:   name,
    Status: process.StatusStopped,
  }

  return c.client.ManageProcess(process)
}

func (c *CLI) handleStart(name string) error {
  return c.client.StartProcess(name)
}

func (c *CLI) handleStop(name string) error {
  return c.client.StopProcess(name)
}

func (c *CLI) handleRestart(name string) error {
  return c.client.RestartProcess(name)
}

func (c *CLI) handleList() error {
  processes, err := c.client.ListProcesses()
  if err != nil {
    return err
  }

  fmt.Println("Managed processes:")
  for _, p := range processes {
    fmt.Printf("Name: %s, Status: %s, PID: %d\n", p.Name, p.Status, p.PID)
  }

  return nil
}

func (c *CLI) handleStatus(name string) error {
  process, err := c.client.GetProcess(name)
  if err != nil {
    return err
  }

  fmt.Printf("Process status for '%s':\n", name)
  fmt.Printf("  Status: %s\n", process.Status)
  fmt.Printf("  PID: %d\n", process.PID)
  fmt.Printf("  Command: %s\n", process.StartCommand)
  fmt.Printf("  Working Directory: %s\n", process.WorkingDir)
  fmt.Printf("  Start Time: %s\n", process.StartTime.Format("2006-01-02 15:04:05"))
  fmt.Printf("  Restart Count: %d\n", process.RestartCount)
  fmt.Printf("  Last Exit Code: %d\n", process.LastExitCode)

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
