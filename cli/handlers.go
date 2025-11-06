package cli

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/casuallc/vigil/config"
  "github.com/casuallc/vigil/inspection"
  "os"
  "os/exec"
  "path/filepath"
  "time"

  "github.com/casuallc/vigil/common"
  "github.com/casuallc/vigil/proc"
  "gopkg.in/yaml.v3" // 导入yaml包用于YAML格式输出
)

// handleBatchScan 处理批量扫描命令
func (c *CLI) handleBatchScan(configFile string, register bool, namespace string) error {
  fmt.Println("Loading scan configuration from:", configFile)

  // 加载配置文件
  scanConfig, err := config.LoadScanConfig(configFile)
  if err != nil {
    return fmt.Errorf("failed to load scan configuration: %v", err)
  }

  fmt.Printf("Found %d processes to scan\n", len(scanConfig.Process))

  // 遍历所有进程配置并执行扫描
  for _, procConfig := range scanConfig.Process {
    fmt.Printf("Scanning for process: %s\n", procConfig.Name)

    // 直接扫描进程
    processes, err := c.client.ScanProcesses(procConfig.Query)
    if err != nil {
      fmt.Printf("Error scanning process %s: %v\n", procConfig.Name, err)
      continue
    }

    // 显示扫描结果
    fmt.Printf("Found %d matching processes for '%s'\n", len(processes), procConfig.Name)

    // 如果需要注册且找到了进程
    if register && len(processes) > 0 {
      // 自动选择第一个匹配的进程并使用配置文件中的名称注册
      selectedProcess := processes[0]
      processName := procConfig.Name // 直接使用配置文件中的名称

      // 创建要注册的进程
      managedProc := proc.ManagedProcess{
        Metadata: selectedProcess.Metadata,
        Spec:     selectedProcess.Spec,
        Status:   selectedProcess.Status,
      }
      managedProc.Metadata.Name = processName
      managedProc.Metadata.Namespace = namespace

      // 添加标签（如果有）
      if len(procConfig.Labels) > 0 {
        fmt.Printf("Adding %d labels to process %s\n", len(procConfig.Labels), processName)
        managedProc.Metadata.Labels = make(map[string]string)
        for _, label := range procConfig.Labels {
          managedProc.Metadata.Labels[label.Name] = label.Value
        }
      }

      // 调用client进行注册
      err = c.client.CreateProcess(managedProc)
      if err != nil {
        fmt.Printf("Failed to register process '%s': %v\n", processName, err)
      } else {
        fmt.Printf("Successfully registered process '%s' in namespace '%s'\n", processName, namespace)
      }
    }
  }

  fmt.Println("Batch scan completed")
  return nil
}

// 修改handleScan函数，显示更多进程信息并支持注册
func (c *CLI) handleScan(query string, registerAfterScan bool, namespace string) error {
  processes, err := c.client.ScanProcesses(query)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
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

  // 修改：不返回错误给 Cobra，打印具体错误原因
  if err := c.client.CreateProcess(process); err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }
  fmt.Printf("Process '%s' created (ns=%s)\n", process.Metadata.Name, process.Metadata.Namespace)
  return nil
}

func (c *CLI) handleStart(name string, namespace string) error {
  if err := c.client.StartProcess(namespace, name); err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }
  fmt.Printf("Process '%s' started (ns=%s)\n", name, namespace)
  return nil
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
  if err := c.client.StopProcess(namespace, name); err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }
  fmt.Printf("Process '%s' stopped (ns=%s)\n", name, namespace)
  return nil
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
  if err := c.client.RestartProcess(namespace, name); err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }
  fmt.Printf("Process '%s' restarted (ns=%s)\n", name, namespace)
  return nil
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
    fmt.Println("ERROR ", err.Error())
    return nil
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
    fmt.Println("ERROR ", err.Error())
    return err
  }

  fmt.Printf("Process status for '%s' in namespace '%s':\n", name, namespace)
  fmt.Printf("  Status: %s\n", process.Status.Phase)
  fmt.Printf("  PID: %d\n", process.Status.PID)
  fmt.Printf("  Command: %s\n", process.Spec.Exec.Command)
  fmt.Printf("  Working Directory: %s\n", process.Spec.WorkingDir)
  if process.Status.StartTime == nil {
    fmt.Printf("  Start Time: N/A\n")
  } else {
    fmt.Printf("  Start Time: %s\n", process.Status.StartTime.Format("2006-01-02 15:04:05"))
  }
  fmt.Printf("  Restart Count: %d\n", process.Status.RestartCount)
  if process.Status.LastTerminationInfo != nil {
    fmt.Printf("  Last Exit Code: %d\n", process.Status.LastTerminationInfo.ExitCode)
  }

  return nil
}

func (c *CLI) handleGet(name string, format string, namespace string) error {
  process, err := c.client.GetProcess(namespace, name)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  if format == "yaml" {
    // 以YAML格式输出进程信息
    yamlData, err := common.ToYamlString(process)
    if err != nil {
      fmt.Println("ERROR failed to marshal proc data to YAML:", err.Error())
      return nil
    }
    fmt.Println(yamlData)
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
      fmt.Printf("  UserGroup: %s\n", process.Spec.UserGroup)
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
  process, err := c.client.GetProcess(namespace, name)
  if err != nil {
    fmt.Println("ERROR failed to get process:", err.Error())
    return nil
  }

  tmpDir, err := os.MkdirTemp("", "vigil-edit-")
  if err != nil {
    fmt.Println("ERROR failed to create temporary directory:", err.Error())
    return nil
  }
  defer func(path string) {
    if rmErr := os.RemoveAll(path); rmErr != nil {
      fmt.Printf("failed to remove temporary directory: %v\n", rmErr)
    }
  }(tmpDir)

  tmpFile := filepath.Join(tmpDir, fmt.Sprintf("%s-%s.yaml", namespace, name))

  yamlData, err := common.ToYamlString(process)
  if err != nil {
    fmt.Println("ERROR failed to marshal process data:", err.Error())
    return nil
  }

  if err := os.WriteFile(tmpFile, []byte(yamlData), 0644); err != nil {
    fmt.Println("ERROR failed to write temporary file:", err.Error())
    return nil
  }

  editor := os.Getenv("EDITOR")
  if editor == "" {
    editor = "vim"
  }

  cmd := exec.Command(editor, tmpFile)
  cmd.Stdin = os.Stdin
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr

  if err := cmd.Run(); err != nil {
    fmt.Println("ERROR editor exited with error:", err.Error())
    return nil
  }

  editedData, err := os.ReadFile(tmpFile)
  if err != nil {
    fmt.Println("ERROR failed to read edited file:", err.Error())
    return nil
  }

  var updatedProc proc.ManagedProcess
  if err := yaml.Unmarshal(editedData, &updatedProc); err != nil {
    fmt.Println("ERROR failed to parse edited data:", err.Error())
    return nil
  }

  updatedProc.Metadata.Name = name
  updatedProc.Metadata.Namespace = namespace

  if err := c.client.UpdateProcess(updatedProc); err != nil {
    fmt.Println("ERROR failed to update process:", err.Error())
    return nil
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

func (c *CLI) handleDelete(name string, namespace string) error {
  if err := c.client.DeleteProcess(namespace, name); err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
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
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Println("System Resource Usage:")
  // System Load
  if resources.Load.Load1 != 0 || resources.Load.Load5 != 0 || resources.Load.Load15 != 0 {
    fmt.Printf("  System Load: 1m=%.2f, 5m=%.2f, 15m=%.2f\n",
      resources.Load.Load1, resources.Load.Load5, resources.Load.Load15)
  }

  // CPU
  fmt.Printf("  CPU Usage: %s\n", resources.CPUUsageHuman)

  // Memory Usage
  if resources.MemoryTotal > 0 {
    fmt.Printf("  Memory Usage: %s / %s (%.2f%%)\n",
      resources.MemoryUsageHuman,
      proc.FormatBytes(resources.MemoryTotal),
      resources.MemoryUsedPercent)
  } else {
    fmt.Printf("  Memory Usage: %s\n", resources.MemoryUsageHuman)
  }

  // Disk Usage (df-style, aligned)
  if len(resources.DiskUsages) > 0 {

    type row struct {
      dev      string
      sizeStr  string
      usedStr  string
      availStr string
      useStr   string
      mount    string
    }

    rows := make([]row, 0, len(resources.DiskUsages))
    devW, sizeW, usedW, availW, useW := len("Filesystem"), len("Size"), len("Used"), len("Avail"), len("Use%")

    for _, du := range resources.DiskUsages {
      if du.Total == 0 {
        continue
      }

      sizeStr := proc.FormatBytes(du.Total)
      usedStr := proc.FormatBytes(du.Used)
      // 使用服务端提供的 Free 作为 Avail
      availVal := du.Free
      if availVal == 0 && du.Total >= du.Used {
        availVal = du.Total - du.Used
      }
      availStr := proc.FormatBytes(availVal)
      useStr := fmt.Sprintf("%d%%", int(du.UsedPercent+0.5))

      r := row{dev: du.Device, sizeStr: sizeStr, usedStr: usedStr, availStr: availStr, useStr: useStr, mount: du.Mountpoint}
      rows = append(rows, r)
      if len(r.dev) > devW {
        devW = len(r.dev)
      }
      if len(r.sizeStr) > sizeW {
        sizeW = len(r.sizeStr)
      }
      if len(r.usedStr) > usedW {
        usedW = len(r.usedStr)
      }
      if len(r.availStr) > availW {
        availW = len(r.availStr)
      }
      if len(r.useStr) > useW {
        useW = len(r.useStr)
      }
    }

    if len(rows) > 0 {
      fmt.Println("  Disk Usage:")
      fmt.Printf("    %-*s %*s %*s %*s %*s %s\n",
        devW, "Filesystem", sizeW, "Size", usedW, "Used", availW, "Avail", useW, "Use%", "Mounted on")
      for _, r := range rows {
        fmt.Printf("    %-*s %*s %*s %*s %*s %s\n",
          devW, r.dev, sizeW, r.sizeStr, usedW, r.usedStr, availW, r.availStr, useW, r.useStr, r.mount)
      }
    }
  }

  // Disk IO (per device)
  if len(resources.DiskIODevices) > 0 {
    fmt.Println("  Disk IO (per device):")
    for _, dio := range resources.DiskIODevices {
      fmt.Printf("    - %s: read=%s, write=%s (reads=%d, writes=%d)\n",
        dio.Device,
        proc.FormatBytes(dio.ReadBytes),
        proc.FormatBytes(dio.WriteBytes),
        dio.ReadCount,
        dio.WriteCount)
    }
  }

  // FD Check
  if resources.FD.Max > 0 {
    fmt.Printf("  File Descriptor: inUse=%d / max=%d (%.2f%%), allocated=%d\n",
      resources.FD.InUse,
      resources.FD.Max,
      resources.FD.UsagePercent,
      resources.FD.CurrentAllocated)
  }

  // Kernel parameter check
  if len(resources.KernelParams) > 0 {
    fmt.Println("  Kernel Parameters:")
    for _, kv := range resources.KernelParams {
      fmt.Printf("    - %s: %s\n", kv.Key, kv.Value)
    }
  }

  // 保留原有总体IO输出（可选）
  fmt.Printf("  Total Disk IO: %s\n", proc.FormatBytes(resources.DiskIO))
  fmt.Printf("  Network IO: %s\n", proc.FormatBytes(resources.NetworkIO))

  return nil
}

func (c *CLI) handleProcessResources(pid int) error {
  resources, err := c.client.GetProcessResources(pid)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Printf("Process Resource Usage for PID %d:\n", pid)

  // CPU 相关
  fmt.Println("  CPU:")
  fmt.Printf("    Usage: %.2f%%\n", resources.CPUUsage)
  if len(resources.CPUAffinity) > 0 {
    fmt.Printf("    Affinity: %v\n", resources.CPUAffinity)
  }
  if resources.CPUUserTime > 0 || resources.CPUSystemTime > 0 || resources.CPUTotalTime > 0 {
    fmt.Printf("    User Time: %s, System Time: %s, Total: %s\n",
      common.FormatSecondsAdaptive(resources.CPUUserTime),
      common.FormatSecondsAdaptive(resources.CPUSystemTime),
      common.FormatSecondsAdaptive(resources.CPUTotalTime))
  }

  // 内存相关
  fmt.Println("  Memory:")
  fmt.Printf("    Virtual (VMS): %s\n", proc.FormatBytes(resources.MemoryVMS))
  fmt.Printf("    Resident (RSS): %s\n", proc.FormatBytes(resources.MemoryRSS))
  if resources.MemoryShared > 0 {
    fmt.Printf("    Shared: %s\n", proc.FormatBytes(resources.MemoryShared))
  }
  if resources.MemoryHeap > 0 {
    fmt.Printf("    Heap: %s\n", proc.FormatBytes(resources.MemoryHeap))
  }

  // I/O 相关
  fmt.Println("  I/O:")
  fmt.Printf("    Read: %s (%d reads)\n", proc.FormatBytes(resources.IOReadBytes), resources.IOReadCount)
  fmt.Printf("    Write: %s (%d writes)\n", proc.FormatBytes(resources.IOWriteBytes), resources.IOWriteCount)
  if resources.IOReadTimeMS > 0 || resources.IOWriteTimeMS > 0 {
    fmt.Printf("    IO Wait: %dms (read), %dms (write)\n", resources.IOReadTimeMS, resources.IOWriteTimeMS)
  }
  if resources.OpenFDs > 0 {
    fmt.Printf("    Open FDs: %d\n", resources.OpenFDs)
  }

  // 进程状态与调度
  fmt.Println("  State & Scheduling:")
  if resources.ProcessStatus != "" {
    fmt.Printf("    Status: %s\n", resources.ProcessStatus)
  }
  if resources.ThreadCount > 0 {
    fmt.Printf("    Threads: %d\n", resources.ThreadCount)
  }
  if resources.CtxSwitchesVoluntary > 0 || resources.CtxSwitchesInvoluntary > 0 {
    fmt.Printf("    Context Switches: voluntary=%d, involuntary=%d\n",
      resources.CtxSwitchesVoluntary, resources.CtxSwitchesInvoluntary)
  }
  if resources.SchedulerPolicy != "" || resources.SchedulerPriority != 0 || resources.Nice != 0 {
    fmt.Printf("    Scheduler: policy=%s, priority=%d, nice=%d\n",
      resources.SchedulerPolicy, resources.SchedulerPriority, resources.Nice)
  }

  // 文件与网络资源
  fmt.Println("  Files & Network:")
  if resources.OpenFilesCount > 0 {
    fmt.Printf("    Open Files: %d\n", resources.OpenFilesCount)
  }
  if resources.NetworkConnectionsCount > 0 {
    fmt.Printf("    Network Connections: %d\n", resources.NetworkConnectionsCount)
  }
  if len(resources.ListeningPorts) > 0 {
    fmt.Println("    Listening Ports:")
    for _, p := range resources.ListeningPorts {
      fmt.Printf("      - %s %s:%d\n", p.Protocol, p.Address, p.Port)
    }
  }

  // 稳定性与生命周期
  fmt.Println("  Lifecycle:")
  if resources.StartTime != nil {
    fmt.Printf("    Start Time: %s\n", resources.StartTime.Format("2006-01-02 15:04:05"))
  }
  if resources.ParentPID > 0 {
    fmt.Printf("    Parent PID: %d\n", resources.ParentPID)
  }
  if resources.ExitCode != 0 {
    fmt.Printf("    Exit Code: %d\n", resources.ExitCode)
  } else {
    fmt.Printf("    Exit Code: N/A\n")
  }

  // 总体IO（保留）
  fmt.Printf("  Total Disk IO: %s\n", proc.FormatBytes(resources.DiskIO))
  fmt.Printf("  Network IO: %s\n", proc.FormatBytes(resources.NetworkIO))

  return nil
}

func (c *CLI) handleGetConfig() error {
  cfg, err := c.client.GetConfig()
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Println("Current Configuration:")
  fmt.Printf("  Log Level: %s\n", cfg.LogLevel)
  fmt.Printf("  Monitor Rate: %d seconds\n", cfg.MonitorRate)
  fmt.Printf("  PID File Path: %s\n", cfg.PidFilePath)
  fmt.Printf("  Managed Apps Count: %d\n", len(cfg.ManagedApps))

  return nil
}

func (c *CLI) handleExec(command string, isFile bool, envVars []string, outputFile string) error {
  // 如果是文件，读取本地文件内容
  if isFile {
    fileContent, err := os.ReadFile(command)
    if err != nil {
      fmt.Println("ERROR failed to read script file:", err.Error())
      return nil
    }
    // 将文件内容作为命令发送，标记为非文件
    command = string(fileContent)
    isFile = false
  }

  // 先执行命令
  output, err := c.client.ExecuteCommand(command, isFile, envVars)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  // 根据outputFile参数决定输出目标（仅在成功时执行）
  if outputFile != "" {
    if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
      fmt.Println("ERROR failed to write output to file:", err.Error())
      return nil
    }
    fmt.Printf("Command output written to %s\n", outputFile)
  } else {
    fmt.Println(output)
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
    fmt.Println("ERROR failed to get process:", err.Error())
    return nil
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

// handleInspection 处理进程巡检命令
func (c *CLI) handleInspection(name string, namespace string, envs []string, configFile string, format string, outputFile string) error {
  // 加载巡检配置文件
  inspectionConfig, err := inspection.LoadInspectionConfig(configFile)
  if err != nil {
    fmt.Printf("ERROR failed to load inspection config: %v\n", err)
    return nil
  }

  // 准备巡检请求
  request := inspection.Request{
    Envs:   envs,
    Config: *inspectionConfig,
  }

  // 执行巡检
  result, err := c.client.InspectProcess(namespace, name, request)
  if err != nil {
    fmt.Printf("ERROR failed to inspect process: %v\n", err)
    return nil
  }

  // 格式化并输出结果
  return c.formatAndOutputInspectionResult(result, format, outputFile)
}

// handleInspectionInteractive 处理交互式进程巡检命令
func (c *CLI) handleInspectionInteractive(namespace string, envs []string, configFile string, format string, outputFile string) error {
  // 获取进程列表
  processes, err := c.client.ListProcesses(namespace)
  if err != nil {
    fmt.Printf("ERROR failed to list processes: %v\n", err)
    return nil
  }

  if len(processes) == 0 {
    fmt.Println("No processes found in namespace:", namespace)
    return nil
  }

  // 准备进程选择列表
  processNames := make([]string, len(processes))
  for i, p := range processes {
    processNames[i] = fmt.Sprintf("%s (Status: %s, PID: %d)", p.Metadata.Name, p.Status.Phase, p.Status.PID)
  }

  // 交互式选择进程
  idx, _, err := Select(SelectConfig{
    Label:    "Select process to inspect",
    Items:    processNames,
    PageSize: 10,
  })
  if err != nil {
    if err.Error() == "user cancelled" {
      fmt.Println("Inspection cancelled.")
      return nil
    }
    fmt.Printf("ERROR selection failed: %v\n", err)
    return nil
  }

  // 获取选中的进程名称
  selectedProcess := processes[idx]

  // 执行巡检
  return c.handleInspection(selectedProcess.Metadata.Name, namespace, envs, configFile, format, outputFile)
}

// formatAndOutputInspectionResult 格式化并输出巡检结果
func (c *CLI) formatAndOutputInspectionResult(result inspection.Result, format string, outputFile string) error {
  var output []byte
  var err error

  // 根据格式选项格式化输出
  switch format {
  case "yaml":
    output, err = yaml.Marshal(result)
    if err != nil {
      return fmt.Errorf("failed to marshal result to yaml: %w", err)
    }
  case "json":
    output, err = json.MarshalIndent(result, "", "  ")
    if err != nil {
      return fmt.Errorf("failed to marshal result to json: %w", err)
    }
  case "text", "":
    // 文本格式输出
    var buf bytes.Buffer
    const lineWidth = 120
    
    // 打印标题和分隔线
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))
    fmt.Fprintf(&buf, "%s\n", centerText("PROCESS INSPECTION REPORT", lineWidth))
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))
    
    // 打印元数据信息（左对齐，冒号对齐）
    fmt.Fprintf(&buf, "%-15s: %s\n", "System", result.Meta.System)
    fmt.Fprintf(&buf, "%-15s: %s\n", "Environment", result.Meta.Env)
    fmt.Fprintf(&buf, "%-15s: %s\n", "Host", result.Meta.Host)
    fmt.Fprintf(&buf, "%-15s: %s\n", "Executed At", result.Meta.ExecutedAt.Format(time.RFC3339))
    fmt.Fprintf(&buf, "%-15s: %.2f seconds\n", "Duration", result.Meta.DurationSeconds)
    fmt.Fprintf(&buf, "%-15s: %s\n", "Status", result.Meta.Status)
    fmt.Fprintf(&buf, "%-15s: %s\n", "Summary", result.Meta.Summary)
    fmt.Fprintf(&buf, "\n")
    
    // 打印检查结果标题
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    fmt.Fprintf(&buf, "%-40s %-12s %-12s %-12s %-40s\n", "Name", "Type", "Status", "Severity", "Message")
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    
    // 打印检查结果（更合理的列宽）
    for _, check := range result.Results {
      fmt.Fprintf(&buf, "%-40s %-12s %-12s %-12s %-40s\n",
        truncateString(check.Name, 40),
        truncateString(check.Type, 12),
        truncateString(check.Status, 12),
        truncateString(check.Severity, 12),
        truncateString(check.Message, 40))
    }
    
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("-"), lineWidth)))
    
    // 打印统计信息（右对齐数字，使冒号对齐）
    fmt.Fprintf(&buf, "%-12s: %6d\n", "Total Checks", result.Summary.TotalChecks)
    fmt.Fprintf(&buf, "%-12s: %6d\n", "OK", result.Summary.OK)
    fmt.Fprintf(&buf, "%-12s: %6d\n", "Warnings", result.Summary.Warn)
    fmt.Fprintf(&buf, "%-12s: %6d\n", "Critical", result.Summary.Critical)
    fmt.Fprintf(&buf, "%-12s: %6d\n", "Errors", result.Summary.Error)
    fmt.Fprintf(&buf, "%-12s: %s\n", "Overall", result.Summary.OverallStatus)
    fmt.Fprintf(&buf, "%s\n", string(bytes.Repeat([]byte("="), lineWidth)))

    output = buf.Bytes()
  default:
    return fmt.Errorf("unsupported output format: %s", format)
  }

  // 输出结果
  if outputFile != "" {
    if err := os.WriteFile(outputFile, output, 0644); err != nil {
      fmt.Printf("ERROR failed to write output to file: %v\n", err)
      return nil
    }
    fmt.Printf("Inspection report written to %s\n", outputFile)
  } else {
    fmt.Println(string(output))
  }

  return nil
}
