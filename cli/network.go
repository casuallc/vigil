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
	"fmt"

	"github.com/casuallc/vigil/api"
	"github.com/spf13/cobra"
)

// setupNetworkCommands 设置所有网络相关的命令
func (c *CLI) setupNetworkCommands() *cobra.Command {
	// Network command - 作为父命令来组织所有网络相关的子命令
	networkCmd := &cobra.Command{
		Use:   "network",
		Short: "Network operations",
		Long:  "Network diagnostic and probing tools",
	}

	// 添加子命令
	networkCmd.AddCommand(c.setupNetworkProbeCommand())

	return networkCmd
}

// setupNetworkProbeCommand 设置 network probe 命令
func (c *CLI) setupNetworkProbeCommand() *cobra.Command {
	var targetIP string
	var port int
	var protocol string
	var timeoutMs int

	probeCmd := &cobra.Command{
		Use:   "probe",
		Short: "Probe a network port",
		Long:  "Probe a network port to check if it's reachable and measure latency",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleNetworkProbe(targetIP, port, protocol, timeoutMs)
		},
	}

	probeCmd.Flags().StringVarP(&targetIP, "target", "t", "", "Target IP address or hostname")
	probeCmd.Flags().IntVarP(&port, "port", "p", 0, "Target port number")
	probeCmd.Flags().StringVar(&protocol, "protocol", "tcp", "Protocol to use (tcp, udp)")
	probeCmd.Flags().IntVar(&timeoutMs, "timeout", 5000, "Timeout in milliseconds")

	probeCmd.MarkFlagRequired("target")
	probeCmd.MarkFlagRequired("port")

	return probeCmd
}

// handleNetworkProbe 处理 network probe 命令
func (c *CLI) handleNetworkProbe(targetIP string, port int, protocol string, timeoutMs int) error {
	req := api.NetworkProbeRequest{
		TargetIP:  targetIP,
		Port:      port,
		Protocol:  protocol,
		TimeoutMs: timeoutMs,
	}

	resp, err := c.client.NetworkProbe(req)
	if err != nil {
		return fmt.Errorf("failed to probe %s:%d: %v", targetIP, port, err)
	}

	if resp.Reachable {
		fmt.Printf("Probe successful: %s:%d is reachable (latency: %dms)\n", targetIP, port, resp.LatencyMs)
	} else {
		fmt.Printf("Probe failed: %s:%d is unreachable (error: %s)\n", targetIP, port, resp.Error)
	}

	return nil
}
