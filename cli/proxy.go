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
	"strings"

	"github.com/casuallc/vigil/api"
	"github.com/casuallc/vigil/proxy"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// setupProxyCommands 设置 HTTP 反向代理管理命令
func (c *CLI) setupProxyCommands() *cobra.Command {
	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "Manage HTTP reverse proxy instances",
		Long:  "Create, start, stop and inspect HTTP reverse proxy instances on the server",
	}

	proxyCmd.AddCommand(c.setupProxyAddCommand())
	proxyCmd.AddCommand(c.setupProxyListCommand())
	proxyCmd.AddCommand(c.setupProxyGetCommand("get", "Show a proxy instance"))
	proxyCmd.AddCommand(c.setupProxyGetCommand("status", "Show a proxy instance's runtime status"))
	proxyCmd.AddCommand(c.setupProxyLifecycleCommand("start", "Start a proxy instance"))
	proxyCmd.AddCommand(c.setupProxyLifecycleCommand("stop", "Stop a proxy instance"))
	proxyCmd.AddCommand(c.setupProxyDeleteCommand())

	return proxyCmd
}

// setupProxyAddCommand 设置 proxy add 命令
func (c *CLI) setupProxyAddCommand() *cobra.Command {
	var name, listen, target, mode string
	var whitelist []string
	var allowPrivate bool
	var maxBodyMB int64
	var headers []string
	var start bool
	var tlsEnabled bool
	var tlsCert, tlsKey string

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Create a proxy instance",
		Long: "Create a proxy instance.\n\n" +
			"Reverse mode (default): forward listen -> target. Whitelist entries:\n" +
			"CIDR (10.0.0.0/8), domain suffix (*.corp.local) or host[:port]. The target\n" +
			"host itself is always allowed.\n\n" +
			"Forward mode (--mode forward): classic forward proxy, the client names the\n" +
			"destination (absolute URI / CONNECT). Clients must authenticate with the\n" +
			"server's super admin credentials via Proxy-Authorization. --whitelist is\n" +
			"required and gates every destination.\n\n" +
			"--allow-private is required for loopback/RFC1918/link-local destinations.\n" +
			"Cloud metadata endpoints are always denied.",
		RunE: func(cmd *cobra.Command, args []string) error {
			headerSet, err := parseHeaderSet(headers)
			if err != nil {
				return err
			}
			if mode == proxy.ModeForward {
				if target != "" {
					return fmt.Errorf("--target is not used in forward mode; destinations come from the client")
				}
				if len(whitelist) == 0 {
					return fmt.Errorf("forward mode requires at least one --whitelist entry (empty = deny all)")
				}
			} else if target == "" {
				return fmt.Errorf("--target is required in reverse mode")
			}
			return c.handleProxyAdd(proxy.InstanceConfig{
				Name:         name,
				Mode:         mode,
				Listen:       listen,
				Target:       target,
				Whitelist:    whitelist,
				AllowPrivate: allowPrivate,
				MaxBodyMB:    maxBodyMB,
				HeaderSet:    headerSet,
				TLS:          proxy.TLSConfig{Enabled: tlsEnabled, CertPath: tlsCert, KeyPath: tlsKey},
			}, start)
		},
	}

	addCmd.Flags().StringVar(&name, "name", "", "Instance name (unique)")
	addCmd.Flags().StringVar(&mode, "mode", "", "Instance mode: reverse (default) | forward")
	addCmd.Flags().StringVar(&listen, "listen", "", "Listen address, e.g. 127.0.0.1:8080")
	addCmd.Flags().StringVar(&target, "target", "", "Reverse mode: upstream target, e.g. http://10.0.0.5:9000")
	addCmd.Flags().StringArrayVar(&whitelist, "whitelist", nil, "Allowed target entry (repeatable)")
	addCmd.Flags().BoolVar(&allowPrivate, "allow-private", false, "Allow loopback/private/link-local targets")
	addCmd.Flags().Int64Var(&maxBodyMB, "max-body-mb", 0, "Max request body in MB (0 = unlimited)")
	addCmd.Flags().StringArrayVar(&headers, "header", nil, "Reverse mode: extra header injected upstream, Key:Value (repeatable)")
	addCmd.Flags().BoolVar(&start, "start", false, "Start the instance immediately")
	addCmd.Flags().BoolVar(&tlsEnabled, "tls", false, "Terminate TLS on the listener")
	addCmd.Flags().StringVar(&tlsCert, "cert", "", "TLS certificate path (with --tls)")
	addCmd.Flags().StringVar(&tlsKey, "key", "", "TLS private key path (with --tls)")

	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("listen")

	return addCmd
}

// setupProxyListCommand 设置 proxy list 命令
func (c *CLI) setupProxyListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List proxy instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleProxyList()
		},
	}
}

// setupProxyGetCommand 设置 proxy get / status 命令
func (c *CLI) setupProxyGetCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [name]",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleProxyGet(args[0])
		},
	}
}

// setupProxyLifecycleCommand 设置 proxy start / stop 命令
func (c *CLI) setupProxyLifecycleCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [name]",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				status *proxy.InstanceStatus
				err    error
			)
			if use == "start" {
				status, err = c.client.ProxyStart(args[0])
			} else {
				status, err = c.client.ProxyStop(args[0])
			}
			if err != nil {
				return fmt.Errorf("failed to %s proxy instance %s: %v", use, args[0], err)
			}
			fmt.Printf("Proxy instance %s: %s\n", status.Name, status.State)
			return nil
		},
	}
}

// setupProxyDeleteCommand 设置 proxy delete 命令
func (c *CLI) setupProxyDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a proxy instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.client.ProxyDelete(args[0]); err != nil {
				return fmt.Errorf("failed to delete proxy instance %s: %v", args[0], err)
			}
			fmt.Printf("Proxy instance %s deleted\n", args[0])
			return nil
		},
	}
}

// handleProxyAdd 处理 proxy add 命令
func (c *CLI) handleProxyAdd(cfg proxy.InstanceConfig, start bool) error {
	status, err := c.client.ProxyCreate(api.ProxyCreateRequest{Config: cfg, Autostart: start})
	if err != nil {
		return fmt.Errorf("failed to create proxy instance %s: %v", cfg.Name, err)
	}
	if status.Config.Mode == proxy.ModeForward {
		fmt.Printf("Proxy instance %s created (%s): forward proxy on %s (whitelist: %s)\n",
			status.Name, status.State, status.Config.Listen, strings.Join(status.Config.Whitelist, ", "))
		return nil
	}
	fmt.Printf("Proxy instance %s created (%s): %s -> %s\n",
		status.Name, status.State, status.Config.Listen, status.Config.Target)
	return nil
}

// handleProxyList 处理 proxy list 命令
func (c *CLI) handleProxyList() error {
	instances, err := c.client.ProxyList()
	if err != nil {
		return fmt.Errorf("failed to list proxy instances: %v", err)
	}
	if len(instances) == 0 {
		fmt.Println("No proxy instances")
		return nil
	}

	table := pterm.TableData{{"NAME", "MODE", "STATE", "ORIGIN", "LISTEN", "TARGET", "REQUESTS", "BYTES OUT"}}
	for _, inst := range instances {
		mode := inst.Config.Mode
		if mode == "" {
			mode = proxy.ModeReverse
		}
		target := inst.Config.Target
		if target == "" {
			target = "-"
		}
		table = append(table, []string{
			inst.Name,
			mode,
			inst.State,
			inst.Origin,
			inst.Config.Listen,
			target,
			fmt.Sprintf("%d", inst.Stats.Requests),
			fmt.Sprintf("%d", inst.Stats.BytesOut),
		})
	}
	return pterm.DefaultTable.WithHasHeader().WithData(table).Render()
}

// handleProxyGet 处理 proxy get / status 命令
func (c *CLI) handleProxyGet(name string) error {
	status, err := c.client.ProxyGet(name)
	if err != nil {
		return fmt.Errorf("failed to get proxy instance %s: %v", name, err)
	}

	fmt.Printf("Name:      %s\n", status.Name)
	mode := status.Config.Mode
	if mode == "" {
		mode = proxy.ModeReverse
	}
	fmt.Printf("Mode:      %s\n", mode)
	fmt.Printf("State:     %s\n", status.State)
	fmt.Printf("Origin:    %s\n", status.Origin)
	fmt.Printf("Listen:    %s\n", status.Config.Listen)
	if status.Config.Target != "" {
		fmt.Printf("Target:    %s\n", status.Config.Target)
	}
	if len(status.Config.Whitelist) > 0 {
		fmt.Printf("Whitelist: %s\n", strings.Join(status.Config.Whitelist, ", "))
	}
	fmt.Printf("Requests:  %d (upstream errors: %d)\n", status.Stats.Requests, status.Stats.UpstreamErr)
	fmt.Printf("Traffic:   in=%dB out=%dB active=%d\n",
		status.Stats.BytesIn, status.Stats.BytesOut, status.Stats.ActiveConn)
	if !status.Stats.StartedAt.IsZero() {
		fmt.Printf("StartedAt: %s\n", status.Stats.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if status.LastError != "" {
		fmt.Printf("LastError: %s\n", status.LastError)
	}
	return nil
}

// parseHeaderSet parses repeated "Key:Value" flags into a header map.
func parseHeaderSet(headers []string) (map[string]string, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		k, v, ok := strings.Cut(h, ":")
		if !ok || strings.TrimSpace(k) == "" {
			return nil, fmt.Errorf("invalid --header %q, want Key:Value", h)
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}
