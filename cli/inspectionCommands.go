package cli

import (
  "github.com/spf13/cobra"
)

// setupInspectionCommand 设置巡检命令
func (c *CLI) setupInspectionCommand() *cobra.Command {
  var inspectionNamespace string
  var inspectionEnvVars []string
  var inspectionConfigFile string
  var inspectionOutputFormat string
  var inspectionOutputFile string

  inspectionCmd := &cobra.Command{
    Use:   "inspect [name]",
    Short: "Inspect process health and status",
    Long: "Run inspection rules against a managed process and generate a detailed report. " +
      "If no name is provided, an interactive selection will be shown.",
    Args: cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {

      // 如果没有提供参数，使用交互式选择
      if len(args) == 0 {
        return c.handleInspectionInteractive(inspectionNamespace, inspectionEnvVars, inspectionConfigFile, inspectionOutputFormat, inspectionOutputFile)
      }

      return c.handleInspection(args[0], inspectionNamespace, inspectionEnvVars, inspectionConfigFile, inspectionOutputFormat, inspectionOutputFile)
    },
  }

  inspectionCmd.Flags().StringVarP(&inspectionNamespace, "namespace", "n", "default", "Process namespace")
  inspectionCmd.Flags().StringArrayVarP(&inspectionEnvVars, "env", "e", []string{}, "Environment variables to set (format: KEY=VALUE)")
  inspectionCmd.Flags().StringVarP(&inspectionConfigFile, "config", "c", "conf/cosmic/rules/admq.yaml", "Inspection rules configuration file")
  inspectionCmd.Flags().StringVarP(&inspectionOutputFormat, "format", "f", "text", "Output format (text|yaml|json)")
  inspectionCmd.Flags().StringVarP(&inspectionOutputFile, "output", "o", "", "Output result to file instead of console")

  inspectionCmd.MarkFlagRequired("config")
  return inspectionCmd
}
