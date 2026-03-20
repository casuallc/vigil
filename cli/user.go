package cli

import (
  "fmt"
  "os"
  "github.com/spf13/cobra"
)

// setupUserCommands sets up all user management related commands
func (c *CLI) setupUserCommands() *cobra.Command {
  userCmd := &cobra.Command{
    Use:   "user",
    Short: "Manage users",
    Long:  "Manage users for the Vigil system",
  }

  // Register subcommands
  userCmd.AddCommand(c.setupUserRegisterCommand())
  userCmd.AddCommand(c.setupUserListCommand())
  userCmd.AddCommand(c.setupUserDeleteCommand())
  userCmd.AddCommand(c.setupUserUpdateCommand())
  userCmd.AddCommand(c.setupUserConfigsCommand())
  userCmd.AddCommand(c.setupUserGetCommand())

  return userCmd
}

// setupUserRegisterCommand sets up the user register command
func (c *CLI) setupUserRegisterCommand() *cobra.Command {
  var username, password, email, role, nickname, avatar, region string

  cmd := &cobra.Command{
    Use:   "register",
    Short: "Register a new user",
    Long:  "Register a new user with username, password, email, and role",
    RunE: func(cmd *cobra.Command, args []string) error {
      if username == "" || password == "" {
        return fmt.Errorf("username and password are required")
      }

      if role == "" {
        role = "user" // default role
      }

      return c.handleUserRegister(username, password, email, role, nickname, avatar, region)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username for the new user")
  cmd.Flags().StringVarP(&password, "password", "p", "", "Password for the new user")
  cmd.Flags().StringVarP(&email, "email", "e", "", "Email for the new user")
  cmd.Flags().StringVarP(&role, "role", "r", "user", "Role for the new user (user/admin)")
  cmd.Flags().StringVarP(&nickname, "nickname", "n", "", "Nickname for the new user")
  cmd.Flags().StringVarP(&avatar, "avatar", "a", "", "Avatar URL for the new user")
  cmd.Flags().StringVarP(&region, "region", "", "", "Region for the new user")

  return cmd
}

// setupUserListCommand sets up the user list command
func (c *CLI) setupUserListCommand() *cobra.Command {
  var verbose bool

  cmd := &cobra.Command{
    Use:   "list",
    Short: "List all users",
    Long:  "List all registered users in the system",
    RunE: func(cmd *cobra.Command, args []string) error {
      return c.handleUserList(verbose)
    },
  }

  cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

  return cmd
}

// setupUserDeleteCommand sets up the user delete command
func (c *CLI) setupUserDeleteCommand() *cobra.Command {
  var username string

  cmd := &cobra.Command{
    Use:   "delete",
    Short: "Delete a user",
    Long:  "Delete a user by username",
    Args:  cobra.NoArgs, // We're using flags instead of positional args
    RunE: func(cmd *cobra.Command, args []string) error {
      if username == "" {
        return fmt.Errorf("username is required")
      }

      return c.handleUserDelete(username)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username to delete")
  _ = cmd.MarkFlagRequired("username")

  return cmd
}

// setupUserUpdateCommand sets up the user update command
func (c *CLI) setupUserUpdateCommand() *cobra.Command {
  var username, email, role, password, nickname, avatar, region string

  cmd := &cobra.Command{
    Use:   "update",
    Short: "Update user information",
    Long:  "Update user information including email, role, and password",
    Args:  cobra.NoArgs, // We're using flags instead of positional args
    RunE: func(cmd *cobra.Command, args []string) error {
      if username == "" {
        return fmt.Errorf("username is required")
      }

      return c.handleUserUpdate(username, email, role, password, nickname, avatar, region)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username to update")
  cmd.Flags().StringVarP(&email, "email", "e", "", "New email for the user")
  cmd.Flags().StringVarP(&role, "role", "r", "", "New role for the user")
  cmd.Flags().StringVarP(&password, "password", "p", "", "New password for the user")
  cmd.Flags().StringVarP(&nickname, "nickname", "n", "", "New nickname for the user")
  cmd.Flags().StringVarP(&avatar, "avatar", "a", "", "New avatar URL for the user")
  cmd.Flags().StringVarP(&region, "region", "", "", "New region for the user")
  _ = cmd.MarkFlagRequired("username")

  return cmd
}

// handleUserRegister handles registering a new user
func (c *CLI) handleUserRegister(username, password, email, role, nickname, avatar, region string) error {
  return c.client.RegisterUserExtended(username, password, email, role, nickname, avatar, region)
}

// handleUserList handles listing all users
func (c *CLI) handleUserList(verbose bool) error {
  users, err := c.client.ListUsers()
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Printf("Found %d users:\n", len(users))
  for _, user := range users {
    if verbose {
      fmt.Printf("- ID: %s, Username: %s, Email: %s, Role: %s, Created: %s\n",
        user.ID, user.Username, user.Email, user.Role, user.CreatedAt.Format("2006-01-02 15:04:05"))
    } else {
      fmt.Printf("- Username: %s, Email: %s, Role: %s\n", user.Username, user.Email, user.Role)
    }
  }

  return nil
}

// handleUserDelete handles deleting a user
func (c *CLI) handleUserDelete(username string) error {
  err := c.client.DeleteUser(username)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Printf("User '%s' deleted successfully!\n", username)
  return nil
}

// handleUserUpdate handles updating a user
func (c *CLI) handleUserUpdate(username, email, role, password, nickname, avatar, region string) error {
  err := c.client.UpdateUserExtended(username, email, role, password, nickname, avatar, region)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Printf("User '%s' updated successfully!\n", username)
  return nil
}

// setupUserConfigsCommand sets up the user configs command
func (c *CLI) setupUserConfigsCommand() *cobra.Command {
  configsCmd := &cobra.Command{
    Use:   "configs",
    Short: "Manage user configurations",
    Long:  "Manage user configurations (get/set)",
  }

  configsCmd.AddCommand(c.setupUserConfigsGetCommand())
  configsCmd.AddCommand(c.setupUserConfigsSetCommand())

  return configsCmd
}

// setupUserConfigsGetCommand sets up the user configs get command
func (c *CLI) setupUserConfigsGetCommand() *cobra.Command {
  var username string

  cmd := &cobra.Command{
    Use:   "get",
    Short: "Get user configuration",
    Long:  "Get user configuration by username",
    RunE: func(cmd *cobra.Command, args []string) error {
      if username == "" {
        return fmt.Errorf("username is required")
      }

      return c.handleUserConfigsGet(username)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username to get configuration")
  _ = cmd.MarkFlagRequired("username")

  return cmd
}

// setupUserConfigsSetCommand sets up the user configs set command
func (c *CLI) setupUserConfigsSetCommand() *cobra.Command {
  var username, configs, configFile string

  cmd := &cobra.Command{
    Use:   "set",
    Short: "Set user configuration",
    Long:  "Set user configuration from string or file",
    RunE: func(cmd *cobra.Command, args []string) error {
      if username == "" {
        return fmt.Errorf("username is required")
      }

      if configs == "" && configFile == "" {
        return fmt.Errorf("either --configs or --file must be provided")
      }

      return c.handleUserConfigsSet(username, configs, configFile)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username to set configuration")
  cmd.Flags().StringVarP(&configs, "configs", "c", "", "Configuration JSON string")
  cmd.Flags().StringVarP(&configFile, "file", "f", "", "Configuration file path")
  _ = cmd.MarkFlagRequired("username")

  return cmd
}

// setupUserGetCommand sets up the user get command
func (c *CLI) setupUserGetCommand() *cobra.Command {
  var username string

  cmd := &cobra.Command{
    Use:   "get",
    Short: "Get user details",
    Long:  "Get user details by username",
    RunE: func(cmd *cobra.Command, args []string) error {
      if username == "" {
        return fmt.Errorf("username is required")
      }

      return c.handleUserGet(username)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username to get details")
  _ = cmd.MarkFlagRequired("username")

  return cmd
}

// handleUserConfigsGet handles getting user configuration
func (c *CLI) handleUserConfigsGet(username string) error {
  configs, err := c.client.GetUserConfigs(username)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  if configs == "" {
    fmt.Printf("User '%s' has no configuration set.\n", username)
  } else {
    fmt.Printf("Configuration for user '%s':\n%s\n", username, configs)
  }

  return nil
}

// handleUserConfigsSet handles setting user configuration
func (c *CLI) handleUserConfigsSet(username, configs, configFile string) error {
  var configData string

  if configFile != "" {
    // Read from file
    data, err := os.ReadFile(configFile)
    if err != nil {
      fmt.Println("ERROR failed to read config file:", err.Error())
      return nil
    }
    configData = string(data)
  } else {
    configData = configs
  }

  err := c.client.UpdateUserConfigs(username, configData)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Printf("User '%s' configuration updated successfully!\n", username)
  return nil
}

// handleUserGet handles getting user details
func (c *CLI) handleUserGet(username string) error {
  user, err := c.client.GetUser(username)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Printf("User Details:\n")
  fmt.Printf("  ID:       %s\n", user.ID)
  fmt.Printf("  Username: %s\n", user.Username)
  fmt.Printf("  Email:    %s\n", user.Email)
  fmt.Printf("  Role:     %s\n", user.Role)
  fmt.Printf("  Nickname: %s\n", user.Nickname)
  fmt.Printf("  Avatar:   %s\n", user.Avatar)
  fmt.Printf("  Region:   %s\n", user.Region)
  fmt.Printf("  Created:  %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
  fmt.Printf("  Updated:  %s\n", user.UpdatedAt.Format("2006-01-02 15:04:05"))

  return nil
}
