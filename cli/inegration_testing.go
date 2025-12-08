package cli

import "github.com/spf13/cobra"

// setupIntegrationTestingCommands 设置测试相关命令
func (c *CLI) setupIntegrationTestingCommands() *cobra.Command {
  testingCmd := &cobra.Command{
    Use:   "test",
    Short: "Run integration tests",
    Long:  "Run integration tests for various services including MQTT",
  }

  // Add MQTT test commands
  mqttTestCmd := c.setupMqttTestCommands()
  testingCmd.AddCommand(mqttTestCmd)

  return testingCmd
}
