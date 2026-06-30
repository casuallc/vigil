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

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/casuallc/vigil/filetransfer"
	"github.com/spf13/cobra"
)

// setupTransferCommands sets up the `transfer` command tree for the
// file-transfer agent.
func (c *CLI) setupTransferCommands() *cobra.Command {
	transferCmd := &cobra.Command{
		Use:   "transfer",
		Short: "File-transfer agent operations",
		Long:  "Browse the filesystem and manage file-transfer tasks on the agent.",
	}
	transferCmd.AddCommand(c.setupTransferFsCommands())
	transferCmd.AddCommand(c.setupTransferTaskCommands())
	return transferCmd
}

// ===================== fs =====================

func (c *CLI) setupTransferFsCommands() *cobra.Command {
	fsCmd := &cobra.Command{
		Use:   "fs",
		Short: "Browse the agent filesystem",
	}

	var listPath string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List a directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleTransferFsList(listPath)
		},
	}
	listCmd.Flags().StringVar(&listPath, "path", "", "Directory path to list")
	listCmd.MarkFlagRequired("path")

	var statPath string
	statCmd := &cobra.Command{
		Use:   "stat",
		Short: "Show file/directory info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleTransferFsStat(statPath)
		},
	}
	statCmd.Flags().StringVar(&statPath, "path", "", "Path to stat")
	statCmd.MarkFlagRequired("path")

	fsCmd.AddCommand(listCmd, statCmd)
	return fsCmd
}

func (c *CLI) handleTransferFsList(path string) error {
	items, err := c.client.FsList(path)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tSIZE\tNAME")
	for _, it := range items {
		kind := "file"
		if it.IsDir {
			kind = "dir"
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", kind, it.Size, it.Name)
	}
	return w.Flush()
}

func (c *CLI) handleTransferFsStat(path string) error {
	stat, err := c.client.FsStat(path)
	if err != nil {
		return err
	}
	return printJSON(stat)
}

// ===================== tasks =====================

func (c *CLI) setupTransferTaskCommands() *cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Manage transfer tasks",
	}

	var createFile string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a task from a TaskConfig JSON file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleTransferTaskCreate(createFile)
		},
	}
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "Path to a TaskConfig JSON file")
	createCmd.MarkFlagRequired("file")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleTransferTaskList()
		},
	}

	taskCmd.AddCommand(createCmd, listCmd)
	taskCmd.AddCommand(
		c.transferIDCommand("get", "Get task config", func(id int64) error {
			task, err := c.client.TransferGetTask(id)
			if err != nil {
				return err
			}
			return printJSON(task)
		}),
		c.transferIDCommand("delete", "Delete a task", func(id int64) error {
			if err := c.client.TransferDeleteTask(id); err != nil {
				return err
			}
			fmt.Printf("Task %d deleted\n", id)
			return nil
		}),
		c.transferIDCommand("start", "Start a task", c.transferLifecycle("started", c.client.TransferStart)),
		c.transferIDCommand("pause", "Pause a task", c.transferLifecycle("paused", c.client.TransferPause)),
		c.transferIDCommand("resume", "Resume a task", c.transferLifecycle("resumed", c.client.TransferResume)),
		c.transferIDCommand("cancel", "Cancel a task", c.transferLifecycle("cancelled", c.client.TransferCancel)),
		c.transferIDCommand("status", "Show task status", func(id int64) error {
			status, err := c.client.TransferStatus(id)
			if err != nil {
				return err
			}
			return printJSON(status)
		}),
		c.transferIDCommand("progress", "Show per-file progress", func(id int64) error {
			progress, err := c.client.TransferProgress(id)
			if err != nil {
				return err
			}
			return printJSON(progress)
		}),
	)
	return taskCmd
}

// transferIDCommand builds a subcommand that requires an --id flag.
func (c *CLI) transferIDCommand(use, short string, run func(id int64) error) *cobra.Command {
	var id int64
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(id)
		},
	}
	cmd.Flags().Int64Var(&id, "id", 0, "Task id")
	cmd.MarkFlagRequired("id")
	return cmd
}

func (c *CLI) transferLifecycle(verb string, action func(int64) error) func(int64) error {
	return func(id int64) error {
		if err := action(id); err != nil {
			return err
		}
		fmt.Printf("Task %d %s\n", id, verb)
		return nil
	}
}

func (c *CLI) handleTransferTaskCreate(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read task file: %w", err)
	}
	var config filetransfer.TaskConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse task config: %w", err)
	}
	id, err := c.client.TransferCreateTask(config)
	if err != nil {
		return err
	}
	fmt.Printf("Task %d created\n", id)
	return nil
}

func (c *CLI) handleTransferTaskList() error {
	tasks, err := c.client.TransferListTasks()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tROLE\tRELAY\tTARGET_DIR")
	for _, t := range tasks {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", t.TaskID, t.Role, t.RelayType, t.TargetDir)
	}
	return w.Flush()
}

func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
