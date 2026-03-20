# CLI User Management Commands

The `bbx-cli user` command group allows managing users in the Vigil system.

## Commands

### `bbx-cli user register`

Registers a new user in the system.

```
bbx-cli user register --username <username> --password <password> [OPTIONS]
```

Options:
- `-u, --username` - Username for the new user (required)
- `-p, --password` - Password for the new user (required)
- `-e, --email` - Email for the new user
- `-r, --role` - Role for the new user (default: user)
- `-n, --nickname` - Nickname for the new user
- `-a, --avatar` - Avatar URL for the new user
- `--region` - Region for the new user

Example:
```
bbx-cli user register -u john -p secure_password -e john@example.com -r user
bbx-cli user register -u john -p secure_password -e john@example.com -r user -n "John Doe" -a "https://example.com/avatar.jpg" --region "Beijing"
```

### `bbx-cli user list`

Lists all users in the system.

```
bbx-cli user list [OPTIONS]
```

Options:
- `-v, --verbose` - Show verbose output with creation dates

Example:
```
bbx-cli user list
bbx-cli user list -v
```

### `bbx-cli user get`

Gets user details.

```
bbx-cli user get --username <username>
```

Options:
- `-u, --username` - Username to get details (required)

Example:
```
bbx-cli user get -u john
```

Output:
```
User Details:
  ID:       usr_1234567890
  Username: john
  Email:    john@example.com
  Role:     user
  Nickname: John Doe
  Avatar:   https://example.com/avatar.jpg
  Region:   Beijing
  Created:  2024-01-01 10:00:00
  Updated:  2024-01-01 10:00:00
```

### `bbx-cli user update`

Updates user information.

```
bbx-cli user update --username <username> [OPTIONS]
```

Options:
- `-u, --username` - Username to update (required)
- `-e, --email` - New email for the user
- `-r, --role` - New role for the user
- `-p, --password` - New password for the user
- `-n, --nickname` - New nickname for the user
- `-a, --avatar` - New avatar URL for the user
- `--region` - New region for the user

Example:
```
bbx-cli user update -u john -e new_email@example.com
bbx-cli user update -u john -p new_password
bbx-cli user update -u john -n "New Name" -a "https://example.com/new-avatar.jpg" --region "Shanghai"
```

### `bbx-cli user delete`

Deletes a user from the system.

```
bbx-cli user delete --username <username>
```

Options:
- `-u, --username` - Username to delete (required)

Example:
```
bbx-cli user delete -u john
```

### `bbx-cli user configs get`

Gets user configuration.

```
bbx-cli user configs get --username <username>
```

Options:
- `-u, --username` - Username to get configuration (required)

Example:
```
bbx-cli user configs get -u john
```

Output:
```
Configuration for user 'john':
{"theme":"dark","language":"zh-CN","notifications":true}
```

Or if no configuration is set:
```
User 'john' has no configuration set.
```

### `bbx-cli user configs set`

Sets user configuration.

```
bbx-cli user configs set --username <username> [--configs <json>] [--file <path>]
```

Options:
- `-u, --username` - Username to set configuration (required)
- `-c, --configs` - Configuration JSON string
- `-f, --file` - Configuration file path

Example:
```
# Set configuration from command line
bbx-cli user configs set -u john -c '{"theme":"dark","language":"zh-CN"}'

# Set configuration from file
bbx-cli user configs set -u john -f ./user-config.json
```

Example config file (user-config.json):
```json
{
  "theme": "dark",
  "language": "zh-CN",
  "notifications": true,
  "customSettings": {
    "fontSize": 14,
    "sidebar": "collapsed"
  }
}
```

## User Profile Fields

The following profile fields are supported:

| Field | Description | Example |
|-------|-------------|---------|
| `avatar` | Avatar URL | `https://example.com/avatar.jpg` |
| `nickname` | User's display name | `John Doe` |
| `region` | User's location/region | `Beijing, China` |
| `configs` | User configuration (JSON string) | `{"theme":"dark"}` |

## Configuration Field

The `configs` field is designed to store large JSON configuration data:

- **Size**: Supports configurations larger than 1MB
- **Format**: JSON string
- **Usage**: Store user preferences, settings, and other custom configurations
- **Access**: Use the `configs get` and `configs set` subcommands

Example configuration structure:
```json
{
  "theme": "dark",
  "language": "zh-CN",
  "notifications": {
    "email": true,
    "push": false,
    "sms": true
  },
  "display": {
    "fontSize": 14,
    "sidebar": "collapsed",
    "compactMode": true
  },
  "shortcuts": {
    "save": "Ctrl+S",
    "open": "Ctrl+O"
  }
}
```

## Authentication

All user management commands require authentication. The default super admin credentials are loaded from `conf/app.conf`, or you can specify different credentials using the `--host` option.

## Authorization

- Super admin and admin users can register, list, update, and delete any users
- Regular users can only update their own information
- Regular users cannot change their role
- Super admin account cannot be deleted
- Users can only access their own configuration (unless admin)
