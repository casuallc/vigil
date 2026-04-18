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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/casuallc/vigil/api"
	"github.com/spf13/cobra"
)

// setupLogCommands configures and returns the "logs" command group.
func (c *CLI) setupLogCommands() *cobra.Command {
	logCmd := &cobra.Command{
		Use:   "logs",
		Short: "Log streaming operations",
		Long:  "Stream and tail log files from the server in real time.",
	}

	logCmd.AddCommand(c.setupLogTailCommand())

	return logCmd
}

// setupLogTailCommand returns the "logs tail" subcommand.
func (c *CLI) setupLogTailCommand() *cobra.Command {
	var path string
	var lines int
	var fromLine int

	tailCmd := &cobra.Command{
		Use:   "tail",
		Short: "Tail a log file in real time",
		Long: `Stream log lines from a file on the server.

Examples:
  bbx-cli logs tail -p /var/log/app.log          # tail from end of file
  bbx-cli logs tail -p /var/log/app.log -n 100   # last 100 lines
  bbx-cli logs tail -p /var/log/app.log -f 0     # from beginning
  bbx-cli logs tail -p /var/log/app.log -f 100   # from line 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleLogTail(path, lines, fromLine)
		},
	}

	tailCmd.Flags().StringVarP(&path, "path", "p", "", "Log file path (required)")
	tailCmd.Flags().IntVarP(&lines, "lines", "n", 0, "Show last N lines (shorthand for --from-line=-N)")
	tailCmd.Flags().IntVarP(&fromLine, "from-line", "f", 0, "Start line number (0=beginning, positive=line num, negative=offset from end)")

	tailCmd.MarkFlagRequired("path")

	return tailCmd
}

// handleLogTail executes the tail logic: connects to the SSE endpoint,
// prints each line to stdout, and blocks until interrupted or the connection closes.
func (c *CLI) handleLogTail(path string, lines, fromLine int) error {
	// --lines takes precedence over --from-line
	if lines > 0 {
		fromLine = -lines
	}

	// Set up signal handling for graceful Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		errCh <- c.client.StreamLogs(ctx, path, fromLine, func(line api.LogLine) {
			fmt.Println(line.Content)
		})
	}()

	select {
	case <-sigCh:
		cancel() // cancel the context to close the HTTP connection
		return nil
	case err := <-errCh:
		return err
	}
}
