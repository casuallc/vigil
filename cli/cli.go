package cli

import (
  "github.com/casuallc/vigil/api"
)

// CLI provides command line interface
type CLI struct {
  client *api.Client
}

// NewCLI creates a new command line interface
func NewCLI(apiHost string) *CLI {
  return &CLI{
    client: api.NewClient(apiHost),
  }
}

// Execute executes command line commands
func (c *CLI) Execute() error {
  return c.setupCommands().Execute()
}
