package cli

import (
  "github.com/casuallc/vigil/client"
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
  return c.setupCommands().Execute()
}
