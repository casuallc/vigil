package cli

import (
  "fmt"
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

  return userCmd
}

// setupUserRegisterCommand sets up the user register command
func (c *CLI) setupUserRegisterCommand() *cobra.Command {
  var username, password, email, role string

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

      return c.handleUserRegister(username, password, email, role)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username for the new user")
  cmd.Flags().StringVarP(&password, "password", "p", "", "Password for the new user")
  cmd.Flags().StringVarP(&email, "email", "e", "", "Email for the new user")
  cmd.Flags().StringVarP(&role, "role", "r", "user", "Role for the new user (user/admin)")

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
  var username, email, role, password string

  cmd := &cobra.Command{
    Use:   "update",
    Short: "Update user information",
    Long:  "Update user information including email, role, and password",
    Args:  cobra.NoArgs, // We're using flags instead of positional args
    RunE: func(cmd *cobra.Command, args []string) error {
      if username == "" {
        return fmt.Errorf("username is required")
      }

      return c.handleUserUpdate(username, email, role, password)
    },
  }

  cmd.Flags().StringVarP(&username, "username", "u", "", "Username to update")
  cmd.Flags().StringVarP(&email, "email", "e", "", "New email for the user")
  cmd.Flags().StringVarP(&role, "role", "r", "", "New role for the user")
  cmd.Flags().StringVarP(&password, "password", "p", "", "New password for the user")
  _ = cmd.MarkFlagRequired("username")

  return cmd
}

// handleUserRegister handles registering a new user
func (c *CLI) handleUserRegister(username, password, email, role string) error {
  return c.client.RegisterUser(username, password, email, role)
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
func (c *CLI) handleUserUpdate(username, email, role, password string) error {
  err := c.client.UpdateUser(username, email, role, password)
  if err != nil {
    fmt.Println("ERROR ", err.Error())
    return nil
  }

  fmt.Printf("User '%s' updated successfully!\n", username)
  return nil
}
