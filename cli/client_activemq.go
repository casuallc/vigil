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
	"path/filepath"

	"github.com/casuallc/vigil/client/activemq"
	"github.com/spf13/cobra"
)

// setupActiveMQCommands sets up ActiveMQ related commands
func (c *CLI) setupActiveMQCommands() *cobra.Command {
	activemqCmd := &cobra.Command{
		Use:   "activemq",
		Short: "ActiveMQ related commands",
		Long:  `Perform ActiveMQ operations like sending and receiving messages via STOMP protocol.`,
	}

	// Add persistent flags to parent command
	var config activemq.ServerConfig
	activemqCmd.PersistentFlags().StringVarP(&config.Server, "server", "s", "127.0.0.1", "ActiveMQ server address")
	activemqCmd.PersistentFlags().IntVarP(&config.Port, "port", "p", 61613, "ActiveMQ STOMP port")
	activemqCmd.PersistentFlags().StringVarP(&config.User, "user", "u", "", "Username for authentication")
	activemqCmd.PersistentFlags().StringVar(&config.Password, "password", "", "Password for authentication")
	activemqCmd.PersistentFlags().IntVar(&config.Timeout, "timeout", 30, "Connection timeout in seconds")
	activemqCmd.PersistentFlags().StringVar(&config.VHost, "vhost", "/", "Virtual host")
	activemqCmd.PersistentFlags().IntVar(&config.HeartBeat, "heartbeat", 0, "Heartbeat interval in seconds (0 to disable)")

	// Store config in context
	activemqCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SetContext(context.WithValue(cmd.Context(), "activemqConfig", &config))
	}

	// Add subcommands
	activemqCmd.AddCommand(c.setupActiveMQSendCommand())
	activemqCmd.AddCommand(c.setupActiveMQReceiveCommand())

	return activemqCmd
}

// setupActiveMQSendCommand sets up the send message command
func (c *CLI) setupActiveMQSendCommand() *cobra.Command {
	var destination string
	var message string
	var filePath string
	var recursive bool
	var repeat int
	var interval int
	var printLog bool
	var headers string
	var persistent bool

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send message to ActiveMQ",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := cmd.Context().Value("activemqConfig").(*activemq.ServerConfig)
			return c.handleActiveMQSend(config, destination, message, filePath, recursive, repeat, interval, printLog, headers, persistent)
		},
	}

	cmd.Flags().StringVarP(&destination, "destination", "d", "", "Message destination (queue or topic, e.g., /queue/test or /topic/test)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message content")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "File or directory path to read message content from")
	cmd.Flags().BoolVarP(&recursive, "recursive", "R", false, "Recursively send files in directory")
	cmd.Flags().IntVarP(&repeat, "repeat", "r", 10, "Number of times to repeat sending")
	cmd.Flags().IntVarP(&interval, "interval", "i", 1000, "Interval between messages in milliseconds")
	cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
	cmd.Flags().StringVar(&headers, "headers", "", "Message headers in format name=value,name2=value2")
	cmd.Flags().BoolVar(&persistent, "persistent", false, "Send as persistent message")

	cmd.MarkFlagRequired("destination")
	cmd.MarkFlagsOneRequired("message", "file")

	return cmd
}

// handleActiveMQSend handles sending messages
func (c *CLI) handleActiveMQSend(config *activemq.ServerConfig, destination, message, filePath string, recursive bool, repeat, interval int, printLog bool, headers string, persistent bool) error {
	client := activemq.NewClient(config)

	if err := client.Connect(); err != nil {
		fmt.Println("ERROR failed to connect to ActiveMQ:", err.Error())
		return nil
	}
	defer client.Close()

	// Directory mode: walk directory, each file as one message
	if filePath != "" {
		info, err := os.Stat(filePath)
		if err != nil {
			fmt.Println("ERROR failed to stat file:", err.Error())
			return nil
		}

		if info.IsDir() {
			sentCount, err := c.sendActiveMQFilesInDir(client, filePath, recursive, destination, interval, printLog, headers, persistent)
			if err != nil {
				fmt.Println("ERROR failed to walk directory:", err.Error())
				return nil
			}
			fmt.Printf("Sent %d messages from directory '%s'\n", sentCount, filePath)
			return nil
		}
	}

	// Single message mode (text or single file)
	producerConfig := &activemq.ProducerConfig{
		Destination: destination,
		Message:     message,
		MessageFile: filePath,
		Repeat:      repeat,
		Interval:    interval,
		PrintLog:    printLog,
		Headers:     headers,
		Persistent:  persistent,
	}

	if err := client.SendMessage(producerConfig); err != nil {
		fmt.Println("ERROR failed to send message:", err.Error())
		return nil
	}

	if filePath != "" {
		fmt.Printf("Successfully sent %d messages from file '%s' to ActiveMQ destination '%s'\n", repeat, filePath, destination)
	} else {
		fmt.Printf("Successfully sent %d messages to ActiveMQ destination '%s'\n", repeat, destination)
	}
	return nil
}

// sendActiveMQFilesInDir walks directory and sends files, each file as one message
func (c *CLI) sendActiveMQFilesInDir(client *activemq.Client, dir string, recursive bool, destination string, interval int, printLog bool, headers string, persistent bool) (int, error) {
	var sentCount int
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == dir {
				return nil
			}
			if !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		producerConfig := &activemq.ProducerConfig{
			Destination: destination,
			MessageFile: path,
			Repeat:      1,
			Interval:    interval,
			PrintLog:    printLog,
			Headers:     headers,
			Persistent:  persistent,
		}

		if err := client.SendMessage(producerConfig); err != nil {
			fmt.Printf("ERROR failed to send file %s: %v\n", path, err)
		} else {
			sentCount++
		}
		return nil
	})
	return sentCount, err
}

// setupActiveMQReceiveCommand sets up the receive message command
func (c *CLI) setupActiveMQReceiveCommand() *cobra.Command {
	var destination string
	var timeout int
	var printLog bool
	var maxMessages int
	var durable bool
	var subscriptionName string
	var ackMode string

	cmd := &cobra.Command{
		Use:   "receive",
		Short: "Receive messages from ActiveMQ",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := cmd.Context().Value("activemqConfig").(*activemq.ServerConfig)
			return c.handleActiveMQReceive(config, destination, timeout, printLog, maxMessages, durable, subscriptionName, ackMode)
		},
	}

	cmd.Flags().StringVarP(&destination, "destination", "d", "", "Message destination to subscribe (e.g., /queue/test or /topic/test)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Consumer timeout in seconds (0 for no timeout)")
	cmd.Flags().BoolVar(&printLog, "print-log", true, "Print detailed logs")
	cmd.Flags().IntVar(&maxMessages, "max-messages", 0, "Maximum number of messages to receive (0 for unlimited)")
	cmd.Flags().BoolVar(&durable, "durable", false, "Create a durable subscription (topics only)")
	cmd.Flags().StringVar(&subscriptionName, "subscription-name", "", "Durable subscription name")
	cmd.Flags().StringVar(&ackMode, "ack-mode", "auto", "Acknowledgment mode (auto, client, client-individual)")

	cmd.MarkFlagRequired("destination")

	return cmd
}

// handleActiveMQReceive handles receiving messages
func (c *CLI) handleActiveMQReceive(config *activemq.ServerConfig, destination string, timeout int, printLog bool, maxMessages int, durable bool, subscriptionName string, ackMode string) error {
	client := activemq.NewClient(config)

	if err := client.Connect(); err != nil {
		fmt.Println("ERROR failed to connect to ActiveMQ:", err.Error())
		return nil
	}
	defer client.Close()

	// Set up consumer config
	consumerConfig := &activemq.ConsumerConfig{
		Destination:      destination,
		Timeout:          timeout,
		PrintLog:         printLog,
		MaxMessages:      maxMessages,
		Durable:          durable,
		SubscriptionName: subscriptionName,
		AckMode:          ackMode,
	}

	// Receive messages
	if err := client.ReceiveMessage(consumerConfig); err != nil {
		fmt.Println("ERROR failed to receive message:", err.Error())
		return nil
	}

	fmt.Printf("Finished receiving messages from ActiveMQ destination '%s'\n", destination)
	return nil
}
