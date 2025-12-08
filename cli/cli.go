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
