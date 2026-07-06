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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/casuallc/vigil/common"
	"github.com/casuallc/vigil/docker"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// setupDockerCommands 设置所有 Docker 相关的命令
func (c *CLI) setupDockerCommands() *cobra.Command {
	dockerCmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker container and compose management",
		Long:  "Manage local Docker containers and compose projects through the vigil server.",
	}

	dockerCmd.AddCommand(c.setupDockerPingCommand())
	dockerCmd.AddCommand(c.setupDockerImageCommands())
	dockerCmd.AddCommand(c.setupDockerContainerCommands())
	dockerCmd.AddCommand(c.setupDockerComposeCommands())

	return dockerCmd
}

// setupDockerImageCommands 设置 docker image 命令组
func (c *CLI) setupDockerImageCommands() *cobra.Command {
	imageCmd := &cobra.Command{
		Use:   "image",
		Short: "Docker image operations",
		Long:  "Download, load, and manage Docker images through the vigil server.",
	}

	imageCmd.AddCommand(c.setupDockerImageLoadCommand())
	imageCmd.AddCommand(c.setupDockerImageListCommand())
	imageCmd.AddCommand(c.setupDockerImageInspectCommand())
	imageCmd.AddCommand(c.setupDockerImagePullCommand())
	imageCmd.AddCommand(c.setupDockerImageRemoveCommand())
	imageCmd.AddCommand(c.setupDockerImageTagCommand())
	imageCmd.AddCommand(c.setupDockerImageHistoryCommand())

	return imageCmd
}

// setupDockerImageLoadCommand 设置 docker image load 命令
func (c *CLI) setupDockerImageLoadCommand() *cobra.Command {
	var url, metadataJSON string

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load a docker image from a remote tar archive",
		Long:  "Download a docker tar archive from a URL and load it into the local docker daemon.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if url == "" {
				return fmt.Errorf("--url is required")
			}

			var metadata docker.ImageMetadata
			if metadataJSON != "" {
				if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
					return fmt.Errorf("invalid metadata JSON: %v", err)
				}
			}

			return c.handleDockerImageLoad(url, metadata)
		},
	}

	cmd.Flags().StringVarP(&url, "url", "u", "", "URL of the docker tar archive")
	cmd.Flags().StringVarP(&metadataJSON, "metadata", "m", "", "Image metadata as JSON string")
	cmd.MarkFlagRequired("url")

	return cmd
}

// handleDockerImageLoad 处理 docker image load 命令
func (c *CLI) handleDockerImageLoad(url string, metadata docker.ImageMetadata) error {
	resp, err := c.client.DockerLoadImage(docker.LoadImageRequest{
		URL:      url,
		Metadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to submit load request: %v", err)
	}

	fmt.Printf("Task created: %s\n", resp.TaskID)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			task, err := c.client.DockerLoadImageStatus(resp.TaskID)
			if err != nil {
				return fmt.Errorf("failed to query task status: %v", err)
			}

			switch task.State {
			case "success":
				fmt.Printf("Load successful. Images: %v\n", task.Images)
				return nil
			case "failed":
				return fmt.Errorf("load failed: %s", task.ErrorMsg)
			default:
				fmt.Printf("Task state: %s\n", task.State)
			}
		}
	}
}

// setupDockerImageListCommand 设置 docker image list 命令
func (c *CLI) setupDockerImageListCommand() *cobra.Command {
	var all, dangling bool
	var labels []string
	var filter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Docker images",
		Long:  "List Docker images in the local daemon.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerImageList(all, dangling, labels, filter)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include intermediate images")
	cmd.Flags().BoolVar(&dangling, "dangling", false, "Filter dangling images")
	cmd.Flags().StringArrayVar(&labels, "filter-label", []string{}, "Filter by label (key=value), can be used multiple times")
	cmd.Flags().StringVar(&filter, "filter", "", "Raw JSON filter string")

	return cmd
}

// handleDockerImageList 处理 docker image list 命令
func (c *CLI) handleDockerImageList(all, dangling bool, labels []string, filter string) error {
	images, err := c.client.DockerListImages(all, dangling, labels, filter)
	if err != nil {
		return fmt.Errorf("failed to list images: %v", err)
	}

	if len(images) == 0 {
		fmt.Println("No images found.")
		return nil
	}

	type imageRow struct {
		id, tag, created, size string
	}
	rows := make([]imageRow, 0, len(images))
	idWidth, tagWidth := len("ID"), len("REPOSITORY:TAG")
	for _, img := range images {
		tag := "<none>:<none>"
		if len(img.RepoTags) > 0 {
			tag = img.RepoTags[0]
		}
		r := imageRow{
			id:      img.ID,
			tag:     tag,
			created: time.Unix(img.Created, 0).Format("2006-01-02 15:04:05"),
			size:    common.FormatFileSize(img.Size),
		}
		rows = append(rows, r)
		if len(r.id) > idWidth {
			idWidth = len(r.id)
		}
		if len(r.tag) > tagWidth {
			tagWidth = len(r.tag)
		}
	}

	fmt.Printf("%-*s %-*s %-20s %s\n", idWidth, "ID", tagWidth, "REPOSITORY:TAG", "CREATED", "SIZE")
	fmt.Println(strings.Repeat("-", idWidth+tagWidth+20+len("SIZE")+3))
	for _, r := range rows {
		fmt.Printf("%-*s %-*s %-20s %s\n",
			idWidth, r.id,
			tagWidth, r.tag,
			r.created,
			r.size,
		)
	}
	return nil
}

// setupDockerImageInspectCommand 设置 docker image inspect 命令
func (c *CLI) setupDockerImageInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect [id]",
		Short: "Inspect a Docker image",
		Long:  "Return detailed information about a Docker image.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerImageInspect(args[0])
		},
	}
}

// handleDockerImageInspect 处理 docker image inspect 命令
func (c *CLI) handleDockerImageInspect(id string) error {
	info, err := c.client.DockerInspectImage(id)
	if err != nil {
		return fmt.Errorf("failed to inspect image: %v", err)
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal image info: %v", err)
	}
	fmt.Println(string(data))
	return nil
}

// setupDockerImagePullCommand 设置 docker image pull 命令
func (c *CLI) setupDockerImagePullCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <image>",
		Short: "Pull a Docker image",
		Long:  "Pull a Docker image from a registry.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerImagePull(args[0])
		},
	}
}

// handleDockerImagePull 处理 docker image pull 命令
func (c *CLI) handleDockerImagePull(image string) error {
	if err := c.client.DockerPullImage(image); err != nil {
		return fmt.Errorf("failed to pull image: %v", err)
	}
	fmt.Printf("Image %s pulled\n", image)
	return nil
}

// setupDockerImageRemoveCommand 设置 docker image rm 命令
func (c *CLI) setupDockerImageRemoveCommand() *cobra.Command {
	var force, noPrune bool

	cmd := &cobra.Command{
		Use:   "rm [id]",
		Short: "Remove a Docker image",
		Long:  "Remove a Docker image from the local daemon.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerImageRemove(args[0], force, noPrune)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal")
	cmd.Flags().BoolVar(&noPrune, "no-prune", false, "Do not delete untagged parent images")

	return cmd
}

// handleDockerImageRemove 处理 docker image rm 命令
func (c *CLI) handleDockerImageRemove(id string, force, noPrune bool) error {
	resp, err := c.client.DockerRemoveImage(id, force, noPrune)
	if err != nil {
		return fmt.Errorf("failed to remove image: %v", err)
	}
	for _, item := range resp.Deleted {
		if item.Deleted != "" {
			fmt.Printf("Deleted: %s\n", item.Deleted)
		}
		if item.Untagged != "" {
			fmt.Printf("Untagged: %s\n", item.Untagged)
		}
	}
	fmt.Printf("Image %s removed\n", id)
	return nil
}

// setupDockerImageTagCommand 设置 docker image tag 命令
func (c *CLI) setupDockerImageTagCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tag <source> <target>",
		Short: "Tag a Docker image",
		Long:  "Create a new tag for an existing Docker image.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerImageTag(args[0], args[1])
		},
	}
}

// handleDockerImageTag 处理 docker image tag 命令
func (c *CLI) handleDockerImageTag(source, target string) error {
	if err := c.client.DockerTagImage(source, target); err != nil {
		return fmt.Errorf("failed to tag image: %v", err)
	}
	fmt.Printf("Tagged %s as %s\n", source, target)
	return nil
}

// setupDockerImageHistoryCommand 设置 docker image history 命令
func (c *CLI) setupDockerImageHistoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "history [id]",
		Short: "Show image history",
		Long:  "Show the history of a Docker image.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerImageHistory(args[0])
		},
	}
}

// handleDockerImageHistory 处理 docker image history 命令
func (c *CLI) handleDockerImageHistory(id string) error {
	history, err := c.client.DockerImageHistory(id)
	if err != nil {
		return fmt.Errorf("failed to get image history: %v", err)
	}

	if len(history) == 0 {
		fmt.Println("No history found.")
		return nil
	}

	type historyRow struct {
		imageID, created, createdBy, size string
	}
	rows := make([]historyRow, 0, len(history))
	imageWidth, createdByWidth := len("IMAGE"), len("CREATED BY")
	for _, h := range history {
		imageID := h.ID
		if imageID == "" {
			imageID = "<missing>"
		}
		r := historyRow{
			imageID:   imageID,
			created:   time.Unix(h.Created, 0).Format("2006-01-02 15:04:05"),
			createdBy: h.CreatedBy,
			size:      common.FormatFileSize(h.Size),
		}
		rows = append(rows, r)
		if len(r.imageID) > imageWidth {
			imageWidth = len(r.imageID)
		}
		if len(r.createdBy) > createdByWidth {
			createdByWidth = len(r.createdBy)
		}
	}

	fmt.Printf("%-*s %-20s %-*s %s\n", imageWidth, "IMAGE", "CREATED", createdByWidth, "CREATED BY", "SIZE")
	fmt.Println(strings.Repeat("-", imageWidth+20+createdByWidth+len("SIZE")+3))
	for _, r := range rows {
		fmt.Printf("%-*s %-20s %-*s %s\n",
			imageWidth, r.imageID,
			r.created,
			createdByWidth, r.createdBy,
			r.size,
		)
	}
	return nil
}

// setupDockerPingCommand 设置 docker ping 命令
func (c *CLI) setupDockerPingCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Ping Docker daemon",
		Long:  "Check connectivity to the Docker daemon through the vigil server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerPing()
		},
	}
}

// setupDockerContainerCommands 设置 docker container 命令组
func (c *CLI) setupDockerContainerCommands() *cobra.Command {
	containerCmd := &cobra.Command{
		Use:   "container",
		Short: "Docker container operations",
		Long:  "List, create, inspect, and manage Docker containers.",
	}

	containerCmd.AddCommand(c.setupDockerContainerListCommand())
	containerCmd.AddCommand(c.setupDockerContainerCreateCommand())
	containerCmd.AddCommand(c.setupDockerContainerInspectCommand())
	containerCmd.AddCommand(c.setupDockerContainerRemoveCommand())
	containerCmd.AddCommand(c.setupDockerContainerStartCommand())
	containerCmd.AddCommand(c.setupDockerContainerStopCommand())
	containerCmd.AddCommand(c.setupDockerContainerRestartCommand())
	containerCmd.AddCommand(c.setupDockerContainerPauseCommand())
	containerCmd.AddCommand(c.setupDockerContainerUnpauseCommand())
	containerCmd.AddCommand(c.setupDockerContainerExecCommand())
	containerCmd.AddCommand(c.setupDockerContainerLogsCommand())
	containerCmd.AddCommand(c.setupDockerContainerStatsCommand())
	containerCmd.AddCommand(c.setupDockerContainerExecWSCommand())
	containerCmd.AddCommand(c.setupDockerContainerLogsWSCommand())

	return containerCmd
}

// setupDockerContainerListCommand 设置 docker container list 命令
func (c *CLI) setupDockerContainerListCommand() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List containers",
		Long:  "List Docker containers managed by the vigil server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerContainerList(all)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include stopped containers")

	return cmd
}

// setupDockerContainerCreateCommand 设置 docker container create 命令
func (c *CLI) setupDockerContainerCreateCommand() *cobra.Command {
	var name, image string
	var cmdSlice, envVars []string
	var ports []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a container",
		Long:  "Create a new Docker container from an image (does not start it).",
		RunE: func(cmd *cobra.Command, args []string) error {
			portMap, err := parseDockerPortMappings(ports)
			if err != nil {
				return err
			}
			return c.handleDockerContainerCreate(name, image, cmdSlice, envVars, portMap)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Container name")
	cmd.Flags().StringVarP(&image, "image", "i", "", "Container image")
	cmd.Flags().StringArrayVarP(&cmdSlice, "cmd", "c", []string{}, "Command arguments (can be used multiple times)")
	cmd.Flags().StringArrayVarP(&envVars, "env", "e", []string{}, "Environment variables KEY=VALUE (can be used multiple times)")
	cmd.Flags().StringArrayVarP(&ports, "port", "p", []string{}, "Port mappings in 'containerPort:hostPort' or 'containerPort:hostIP:hostPort' format")

	cmd.MarkFlagRequired("image")

	return cmd
}

// setupDockerContainerInspectCommand 设置 docker container inspect 命令
func (c *CLI) setupDockerContainerInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect [id]",
		Short: "Inspect a container",
		Long:  "Return detailed information about a Docker container. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to inspect")
			if err != nil {
				return err
			}
			return c.handleDockerContainerInspect(id)
		},
	}
}

// setupDockerContainerRemoveCommand 设置 docker container rm 命令
func (c *CLI) setupDockerContainerRemoveCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rm [id]",
		Short: "Remove a container",
		Long:  "Remove a Docker container. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to remove")
			if err != nil {
				return err
			}
			return c.handleDockerContainerRemove(id, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force remove a running container")

	return cmd
}

// setupDockerContainerStartCommand 设置 docker container start 命令
func (c *CLI) setupDockerContainerStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start [id]",
		Short: "Start a container",
		Long:  "Start a Docker container. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to start")
			if err != nil {
				return err
			}
			return c.handleDockerContainerStart(id)
		},
	}
}

// setupDockerContainerStopCommand 设置 docker container stop 命令
func (c *CLI) setupDockerContainerStopCommand() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "stop [id]",
		Short: "Stop a container",
		Long:  "Stop a Docker container with an optional timeout. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to stop")
			if err != nil {
				return err
			}
			return c.handleDockerContainerStop(id, timeout)
		},
	}

	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "Graceful stop timeout in seconds (0 uses daemon default)")

	return cmd
}

// setupDockerContainerRestartCommand 设置 docker container restart 命令
func (c *CLI) setupDockerContainerRestartCommand() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "restart [id]",
		Short: "Restart a container",
		Long:  "Restart a Docker container with an optional timeout. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to restart")
			if err != nil {
				return err
			}
			return c.handleDockerContainerRestart(id, timeout)
		},
	}

	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "Graceful stop timeout in seconds (0 uses daemon default)")

	return cmd
}

// setupDockerContainerPauseCommand 设置 docker container pause 命令
func (c *CLI) setupDockerContainerPauseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pause [id]",
		Short: "Pause a container",
		Long:  "Pause all processes in a Docker container. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to pause")
			if err != nil {
				return err
			}
			return c.handleDockerContainerPause(id)
		},
	}
}

// setupDockerContainerUnpauseCommand 设置 docker container unpause 命令
func (c *CLI) setupDockerContainerUnpauseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unpause [id]",
		Short: "Unpause a container",
		Long:  "Resume all processes in a Docker container. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to unpause")
			if err != nil {
				return err
			}
			return c.handleDockerContainerUnpause(id)
		},
	}
}

// setupDockerContainerExecCommand 设置 docker container exec 命令
func (c *CLI) setupDockerContainerExecCommand() *cobra.Command {
	var command string
	var tty, interactive bool

	cmd := &cobra.Command{
		Use:   "exec [id]",
		Short: "Execute a command in a container",
		Long:  "Run a command inside a Docker container. Use -i/-t for an interactive session (e.g. bash). If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to exec")
			if err != nil {
				return err
			}
			if command == "" {
				return fmt.Errorf("command is required")
			}
			return c.handleDockerContainerExec(id, command, tty, interactive)
		},
	}

	cmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute")
	cmd.Flags().BoolVarP(&tty, "tty", "t", false, "Allocate a pseudo-TTY")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Keep STDIN open for interactive commands")
	cmd.MarkFlagRequired("command")

	return cmd
}

// setupDockerContainerLogsCommand 设置 docker container logs 命令
func (c *CLI) setupDockerContainerLogsCommand() *cobra.Command {
	var follow bool
	var tail, since string

	cmd := &cobra.Command{
		Use:   "logs [id]",
		Short: "Fetch container logs",
		Long:  "Stream or fetch logs from a Docker container. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to fetch logs")
			if err != nil {
				return err
			}
			return c.handleDockerContainerLogs(id, follow, tail, since)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().StringVar(&tail, "tail", "", "Output the last N lines")
	cmd.Flags().StringVar(&since, "since", "", "Show logs since timestamp")

	return cmd
}

// setupDockerContainerStatsCommand 设置 docker container stats 命令
func (c *CLI) setupDockerContainerStatsCommand() *cobra.Command {
	var stream bool

	cmd := &cobra.Command{
		Use:   "stats [id]",
		Short: "Fetch container stats",
		Long:  "Stream or fetch resource usage statistics from a Docker container. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to fetch stats")
			if err != nil {
				return err
			}
			return c.handleDockerContainerStats(id, stream)
		},
	}

	cmd.Flags().BoolVarP(&stream, "stream", "s", true, "Stream stats continuously")

	return cmd
}

// setupDockerContainerExecWSCommand 设置 docker container exec-ws 命令
func (c *CLI) setupDockerContainerExecWSCommand() *cobra.Command {
	var command string
	var tty bool
	var width, height int

	cmd := &cobra.Command{
		Use:   "exec-ws [id]",
		Short: "Interactive exec via WebSocket",
		Long:  "Open an interactive exec session inside a Docker container via WebSocket. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to exec")
			if err != nil {
				return err
			}
			if command == "" {
				return fmt.Errorf("command is required")
			}
			return c.handleDockerContainerExecWS(id, command, tty, width, height)
		},
	}

	cmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute interactively")
	cmd.Flags().BoolVarP(&tty, "tty", "t", true, "Allocate a pseudo-TTY")
	cmd.Flags().IntVarP(&width, "width", "W", 120, "Terminal width")
	cmd.Flags().IntVar(&height, "height", 40, "Terminal height")
	cmd.MarkFlagRequired("command")

	return cmd
}

// setupDockerContainerLogsWSCommand 设置 docker container logs-ws 命令
func (c *CLI) setupDockerContainerLogsWSCommand() *cobra.Command {
	var tail, since string

	cmd := &cobra.Command{
		Use:   "logs-ws [id]",
		Short: "Stream container logs via WebSocket",
		Long:  "Stream logs from a Docker container via WebSocket until interrupted. If no id is provided, an interactive selection will be shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := c.resolveDockerContainerID(args, "select container to stream logs")
			if err != nil {
				return err
			}
			return c.handleDockerContainerLogsWS(id, tail, since)
		},
	}

	cmd.Flags().StringVar(&tail, "tail", "", "Output the last N lines")
	cmd.Flags().StringVar(&since, "since", "", "Show logs since timestamp")

	return cmd
}

// setupDockerComposeCommands 设置 docker compose 命令组
func (c *CLI) setupDockerComposeCommands() *cobra.Command {
	composeCmd := &cobra.Command{
		Use:   "compose",
		Short: "Docker Compose operations",
		Long:  "Deploy, inspect, and remove Docker Compose projects.",
	}

	composeCmd.AddCommand(c.setupDockerComposeUpCommand())
	composeCmd.AddCommand(c.setupDockerComposeStatusCommand())
	composeCmd.AddCommand(c.setupDockerComposeDownCommand())

	return composeCmd
}

// setupDockerComposeUpCommand 设置 docker compose up 命令
func (c *CLI) setupDockerComposeUpCommand() *cobra.Command {
	var file, name string
	var start bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Deploy a compose project",
		Long:  "Deploy a Docker Compose project defined by a docker-compose.yml file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerComposeUp(file, name, start)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "docker-compose.yml", "Path to docker-compose.yml file")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Project name")
	cmd.Flags().BoolVar(&start, "start", true, "Start containers after creation")

	cmd.MarkFlagRequired("name")

	return cmd
}

// setupDockerComposeStatusCommand 设置 docker compose status 命令
func (c *CLI) setupDockerComposeStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status [project]",
		Short: "Show compose project status",
		Long:  "Display the status of all services in a Docker Compose project.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerComposeStatus(args[0])
		},
	}
}

// setupDockerComposeDownCommand 设置 docker compose down 命令
func (c *CLI) setupDockerComposeDownCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "down [project]",
		Short: "Remove a compose project",
		Long:  "Stop and remove all containers, networks, and volumes of a compose project.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleDockerComposeDown(args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force remove running containers")

	return cmd
}

// ------------------------- Command Handlers -------------------------

// handleDockerPing 处理 docker ping 命令
func (c *CLI) handleDockerPing() error {
	ping, err := c.client.DockerPing()
	if err != nil {
		return fmt.Errorf("failed to ping docker daemon: %v", err)
	}
	fmt.Printf("Docker daemon is reachable\n")
	fmt.Printf("  API Version:  %s\n", ping.APIVersion)
	fmt.Printf("  OS Type:      %s\n", ping.OSType)
	fmt.Printf("  Experimental: %v\n", ping.Experimental)
	return nil
}

// handleDockerContainerList 处理 docker container list 命令
func (c *CLI) handleDockerContainerList(all bool) error {
	containers, err := c.client.DockerListContainers(all)
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	if len(containers) == 0 {
		fmt.Println("No containers found.")
		return nil
	}

	type containerRow struct {
		id, image, names, state, status, ports string
	}
	rows := make([]containerRow, 0, len(containers))
	idWidth, imageWidth, namesWidth, stateWidth, statusWidth := len("ID"), len("IMAGE"), len("NAMES"), len("STATE"), len("STATUS")
	for _, ct := range containers {
		r := containerRow{
			id:      ct.ID,
			image:   ct.Image,
			names:   strings.Join(ct.Names, ", "),
			state:   ct.State,
			status:  ct.Status,
			ports:   formatDockerPorts(ct.Ports),
		}
		rows = append(rows, r)
		if len(r.id) > idWidth {
			idWidth = len(r.id)
		}
		if len(r.image) > imageWidth {
			imageWidth = len(r.image)
		}
		if len(r.names) > namesWidth {
			namesWidth = len(r.names)
		}
		if len(r.state) > stateWidth {
			stateWidth = len(r.state)
		}
		if len(r.status) > statusWidth {
			statusWidth = len(r.status)
		}
	}

	fmt.Printf("%-*s %-*s %-*s %-*s %-*s %s\n",
		idWidth, "ID",
		imageWidth, "IMAGE",
		namesWidth, "NAMES",
		stateWidth, "STATE",
		statusWidth, "STATUS",
		"PORTS")
	fmt.Println(strings.Repeat("-", idWidth+imageWidth+namesWidth+stateWidth+statusWidth+len("PORTS")+5))
	for _, r := range rows {
		fmt.Printf("%-*s %-*s %-*s %-*s %-*s %s\n",
			idWidth, r.id,
			imageWidth, r.image,
			namesWidth, r.names,
			stateWidth, r.state,
			statusWidth, r.status,
			r.ports,
		)
	}

	return nil
}

// selectDockerContainerInteractively 交互式选择已有容器
func (c *CLI) selectDockerContainerInteractively(label string) (docker.ContainerSummary, error) {
	containers, err := c.client.DockerListContainers(true)
	if err != nil {
		return docker.ContainerSummary{}, fmt.Errorf("failed to list containers: %v", err)
	}

	if len(containers) == 0 {
		return docker.ContainerSummary{}, fmt.Errorf("no containers found")
	}

	options := make([]SelectOption, len(containers))
	for i, ct := range containers {
		names := strings.Join(ct.Names, ", ")
		if names == "" {
			names = "<none>"
		}
		options[i] = SelectOption{
			Value: ct.ID,
			Label: fmt.Sprintf("%s (%s, %s, %s)", truncateString(ct.ID, 12), ct.Image, names, ct.Status),
		}
	}

	idx, _, err := Select(SelectConfig{
		Label: label,
		Items: options,
	})
	if err != nil {
		return docker.ContainerSummary{}, err
	}
	return containers[idx], nil
}

// resolveDockerContainerID 根据命令行参数或交互式选择解析容器 ID
func (c *CLI) resolveDockerContainerID(args []string, label string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	ct, err := c.selectDockerContainerInteractively(label)
	if err != nil {
		return "", err
	}
	return ct.ID, nil
}

// handleDockerContainerCreate 处理 docker container create 命令
func (c *CLI) handleDockerContainerCreate(name, image string, cmd, env []string, ports map[string]string) error {
	id, err := c.client.DockerCreateContainer(name, image, cmd, env, ports)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}
	fmt.Printf("Container created: %s\n", id)
	return nil
}

// handleDockerContainerInspect 处理 docker container inspect 命令
func (c *CLI) handleDockerContainerInspect(id string) error {
	info, err := c.client.DockerInspectContainer(id)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %v", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal container info: %v", err)
	}
	fmt.Println(string(data))
	return nil
}

// handleDockerContainerRemove 处理 docker container rm 命令
func (c *CLI) handleDockerContainerRemove(id string, force bool) error {
	if err := c.client.DockerRemoveContainer(id, force); err != nil {
		return fmt.Errorf("failed to remove container: %v", err)
	}
	fmt.Printf("Container %s removed\n", id)
	return nil
}

// handleDockerContainerStart 处理 docker container start 命令
func (c *CLI) handleDockerContainerStart(id string) error {
	if err := c.client.DockerStartContainer(id); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}
	fmt.Printf("Container %s started\n", id)
	return nil
}

// handleDockerContainerStop 处理 docker container stop 命令
func (c *CLI) handleDockerContainerStop(id string, timeout int) error {
	var t *int
	if timeout > 0 {
		t = &timeout
	}
	if err := c.client.DockerStopContainer(id, t); err != nil {
		return fmt.Errorf("failed to stop container: %v", err)
	}
	fmt.Printf("Container %s stopped\n", id)
	return nil
}

// handleDockerContainerRestart 处理 docker container restart 命令
func (c *CLI) handleDockerContainerRestart(id string, timeout int) error {
	var t *int
	if timeout > 0 {
		t = &timeout
	}
	if err := c.client.DockerRestartContainer(id, t); err != nil {
		return fmt.Errorf("failed to restart container: %v", err)
	}
	fmt.Printf("Container %s restarted\n", id)
	return nil
}

// handleDockerContainerPause 处理 docker container pause 命令
func (c *CLI) handleDockerContainerPause(id string) error {
	if err := c.client.DockerPauseContainer(id); err != nil {
		return fmt.Errorf("failed to pause container: %v", err)
	}
	fmt.Printf("Container %s paused\n", id)
	return nil
}

// handleDockerContainerUnpause 处理 docker container unpause 命令
func (c *CLI) handleDockerContainerUnpause(id string) error {
	if err := c.client.DockerUnpauseContainer(id); err != nil {
		return fmt.Errorf("failed to unpause container: %v", err)
	}
	fmt.Printf("Container %s unpaused\n", id)
	return nil
}

// handleDockerContainerExec 处理 docker container exec 命令
func (c *CLI) handleDockerContainerExec(id, command string, tty, interactive bool) error {
	if interactive || tty {
		// Interactive commands need a bidirectional stream. Fall back to the
		// WebSocket exec path so stdin/stdout are forwarded in real time.
		if interactive {
			tty = true
		}
		width, height := 120, 40
		if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			width, height = w, h
		}
		conn, err := c.client.DockerExecWebSocket(id, command, tty, width, height)
		if err != nil {
			return fmt.Errorf("failed to open interactive exec: %v", err)
		}
		defer conn.Close()
		return dockerWebSocketInteractive(conn)
	}

	output, err := c.client.DockerExecContainer(id, command, tty)
	if err != nil {
		return fmt.Errorf("failed to exec command: %v", err)
	}
	fmt.Print(output)
	return nil
}

// handleDockerContainerLogs 处理 docker container logs 命令
func (c *CLI) handleDockerContainerLogs(id string, follow bool, tail, since string) error {
	reader, err := c.client.DockerStreamLogs(id, follow, tail, since)
	if err != nil {
		return fmt.Errorf("failed to stream logs: %v", err)
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("failed to read logs: %v", err)
	}
	return nil
}

// handleDockerContainerStats 处理 docker container stats 命令
func (c *CLI) handleDockerContainerStats(id string, stream bool) error {
	reader, err := c.client.DockerStreamStats(id, stream)
	if err != nil {
		return fmt.Errorf("failed to stream stats: %v", err)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read stats: %v", err)
	}
	return nil
}

// handleDockerContainerExecWS 处理 docker container exec-ws 命令
func (c *CLI) handleDockerContainerExecWS(id, command string, tty bool, width, height int) error {
	conn, err := c.client.DockerExecWebSocket(id, command, tty, width, height)
	if err != nil {
		return fmt.Errorf("failed to open exec websocket: %v", err)
	}
	defer conn.Close()

	return dockerWebSocketInteractive(conn)
}

// handleDockerContainerLogsWS 处理 docker container logs-ws 命令
func (c *CLI) handleDockerContainerLogsWS(id, tail, since string) error {
	conn, err := c.client.DockerLogsWebSocket(id, tail, since)
	if err != nil {
		return fmt.Errorf("failed to open logs websocket: %v", err)
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return nil
			}
			return fmt.Errorf("websocket read error: %v", err)
		}
		fmt.Print(string(message))
	}
}

// handleDockerComposeUp 处理 docker compose up 命令
func (c *CLI) handleDockerComposeUp(file, name string, start bool) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read compose file %s: %v", file, err)
	}

	startPtr := &start
	status, err := c.client.DockerComposeDeploy(docker.ComposeDeployRequest{
		Name:    name,
		Content: string(content),
		Start:   startPtr,
	})
	if err != nil {
		return fmt.Errorf("failed to deploy compose project: %v", err)
	}

	fmt.Printf("Project %s deployed\n", status.Name)
	printComposeProjectStatus(status)
	return nil
}

// handleDockerComposeStatus 处理 docker compose status 命令
func (c *CLI) handleDockerComposeStatus(project string) error {
	status, err := c.client.DockerComposeGet(project)
	if err != nil {
		return fmt.Errorf("failed to get compose project status: %v", err)
	}

	printComposeProjectStatus(status)
	return nil
}

// handleDockerComposeDown 处理 docker compose down 命令
func (c *CLI) handleDockerComposeDown(project string, force bool) error {
	if err := c.client.DockerComposeRemove(project, force); err != nil {
		return fmt.Errorf("failed to remove compose project: %v", err)
	}
	fmt.Printf("Project %s removed\n", project)
	return nil
}

// ------------------------- Helpers -------------------------

// parseDockerPortMappings 将命令行端口映射参数解析为 API 格式
func parseDockerPortMappings(ports []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, p := range ports {
		parts := strings.Split(p, ":")
		switch len(parts) {
		case 2:
			// containerPort:hostPort
			result[parts[0]] = parts[1]
		case 3:
			// containerPort:hostIP:hostPort
			result[parts[0]] = parts[1] + ":" + parts[2]
		default:
			return nil, fmt.Errorf("invalid port mapping %q (expected containerPort:hostPort or containerPort:hostIP:hostPort)", p)
		}
	}
	return result, nil
}

// formatDockerPorts 格式化端口映射为可读字符串
func formatDockerPorts(ports []docker.PortMapping) string {
	if len(ports) == 0 {
		return "-"
	}
	var parts []string
	for _, p := range ports {
		if p.IP != "" {
			parts = append(parts, fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type))
		} else {
			parts = append(parts, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
}

// printComposeProjectStatus 打印 compose 项目状态
func printComposeProjectStatus(status *docker.ComposeProjectStatus) {
	fmt.Printf("Project: %s\n", status.Name)
	if len(status.Services) == 0 {
		fmt.Println("  No services found.")
		return
	}
	for _, svc := range status.Services {
		fmt.Printf("  Service: %s (image: %s, replicas: %d)\n", svc.Name, svc.Image, svc.Replicas)
		for _, ct := range svc.Containers {
			fmt.Printf("    %-14s %-20s %-12s %-20s\n",
				truncateString(ct.ID, 14),
				truncateString(ct.Image, 20),
				ct.State,
				ct.Status,
			)
		}
	}
}

// dockerWebSocketInteractive 通过 WebSocket 建立交互式会话
func dockerWebSocketInteractive(conn *websocket.Conn) error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var (
		connClosed   bool
		mu           sync.Mutex
		lastW, lastH int
	)

	// 发送初始窗口大小
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		lastW, lastH = w, h
		_ = sendDockerResize(conn, &connClosed, &mu, w, h)
	}

	// 窗口大小变化时发送 resize 消息
	go func() {
		for {
			mu.Lock()
			if connClosed {
				mu.Unlock()
				return
			}
			mu.Unlock()

			w, h, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if w == lastW && h == lastH {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			lastW, lastH = w, h
			if err := sendDockerResize(conn, &connClosed, &mu, w, h); err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 读取 WebSocket 消息并输出到终端
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				mu.Lock()
				connClosed = true
				mu.Unlock()
				term.Restore(int(os.Stdin.Fd()), oldState)
				os.Exit(0)
			}
			os.Stdout.Write(message)
		}
	}()

	// 从标准输入读取并发送到 WebSocket
	buffer := make([]byte, 1024)
	for {
		mu.Lock()
		if connClosed {
			mu.Unlock()
			return nil
		}
		mu.Unlock()

		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return nil
		}
		if n > 0 {
			if err := writeWebSocketMessage(conn, &connClosed, &mu, websocket.TextMessage, buffer[:n]); err != nil {
				return nil
			}
		}
	}
}

// sendDockerResize 发送终端 resize 消息
func sendDockerResize(conn *websocket.Conn, connClosed *bool, mu *sync.Mutex, w, h int) error {
	resizeMsg := map[string]int{"cols": w, "rows": h}
	resizeJSON, _ := json.Marshal(resizeMsg)
	return writeWebSocketMessage(conn, connClosed, mu, websocket.TextMessage, []byte("resize:"+string(resizeJSON)))
}
