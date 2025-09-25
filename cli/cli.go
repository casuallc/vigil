package cli

import (
	"fmt"
	"github.com/casuallc/vigil/client"
	"github.com/casuallc/vigil/process"
	"github.com/spf13/cobra"
)

// CLI provides command line interface
type CLI struct {
	client *client.Client
}

// NewCLI creates a new command line interface
func NewCLI(apiHost string) *CLI {
	return &CLI{
		client: client.NewClient(apiHost),
	}
}

// Execute executes command line commands
func (c *CLI) Execute() error {
	// Root command
	var apiHost string

	rootCmd := &cobra.Command{
		Use:     "vigil",
		Short:   "Process Management Program",
		Long:    "Vigil - A powerful process management and monitoring tool",
		Version: "1.0.0",
	}

	// Global flag for API host
	rootCmd.PersistentFlags().StringVarP(&apiHost, "host", "H", "http://localhost:8080", "API server host address")

	// Override PreRun to create client with the provided host
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Only create a new client if we're not in the root command
		if cmd != rootCmd {
			c.client = client.NewClient(apiHost)
		}
		return nil
	}

	// Scan command
	var query string
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan processes",
		Long:  "Scan system processes based on query string or regex",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleScan(query)
		},
	}
	scanCmd.Flags().StringVarP(&query, "query", "q", "", "Search query string or regex")
	scanCmd.MarkFlagRequired("query")

	// Manage command
	var (
		processName string
		commandPath string
	)
	manageCmd := &cobra.Command{
		Use:   "manage",
		Short: "Manage process",
		Long:  "Manage an existing process or create a new managed process",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleManage(processName, commandPath)
		},
	}
	manageCmd.Flags().StringVarP(&processName, "name", "n", "", "Process name")
	manageCmd.Flags().StringVarP(&commandPath, "command", "c", "", "Command path")
	manageCmd.MarkFlagRequired("name")

	// Start command
	var startName string
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start process",
		Long:  "Start a managed process",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleStart(startName)
		},
	}
	startCmd.Flags().StringVarP(&startName, "name", "n", "", "Process name")
	startCmd.MarkFlagRequired("name")

	// Stop command
	var stopName string
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop process",
		Long:  "Stop a managed process",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleStop(stopName)
		},
	}
	stopCmd.Flags().StringVarP(&stopName, "name", "n", "", "Process name")
	stopCmd.MarkFlagRequired("name")

	// Restart command
	var restartName string
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart process",
		Long:  "Restart a managed process",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleRestart(restartName)
		},
	}
	restartCmd.Flags().StringVarP(&restartName, "name", "n", "", "Process name")
	restartCmd.MarkFlagRequired("name")

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List processes",
		Long:  "List all managed processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleList()
		},
	}

	// Status command
	var statusName string
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check process status",
		Long:  "Check the status of a managed process",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleStatus(statusName)
		},
	}
	statusCmd.Flags().StringVarP(&statusName, "name", "n", "", "Process name")
	statusCmd.MarkFlagRequired("name")

	// Resource commands
	systemResourceCmd := &cobra.Command{
		Use:   "system-resources",
		Short: "Get system resources",
		Long:  "Get system resource usage information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleSystemResources()
		},
	}

	processResourceCmd := &cobra.Command{
		Use:   "process-resources",
		Short: "Get process resources",
		Long:  "Get resource usage information for a specific process",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := 0
			if len(args) > 0 {
				fmt.Sscanf(args[0], "%d", &pid)
			}
			return c.handleProcessResources(pid)
		},
	}

	// Config commands
	getConfigCmd := &cobra.Command{
		Use:   "get-config",
		Short: "Get configuration",
		Long:  "Get the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleGetConfig()
		},
	}

	// Add commands to root command
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(manageCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(systemResourceCmd)
	rootCmd.AddCommand(processResourceCmd)
	rootCmd.AddCommand(getConfigCmd)

	// Execute the root command
	return rootCmd.Execute()
}

// Command handlers
func (c *CLI) handleScan(query string) error {
	processes, err := c.client.ScanProcesses(query)
	if err != nil {
		return err
	}

	fmt.Println("Scanned processes:")
	for _, p := range processes {
		fmt.Printf("PID: %d, Name: %s\n", p.PID, p.Name)
	}

	return nil
}

func (c *CLI) handleManage(name, command string) error {
	process := process.ManagedProcess{
		Name:    name,
		Command: command,
		Status:  process.StatusStopped,
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
	fmt.Printf("  Command: %s\n", process.Command)
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
