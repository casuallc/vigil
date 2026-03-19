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
- `-r, --role` - Role for the new user (default: user, can be admin)

Example:
```
bbx-cli user register -u john -p secure_password -e john@example.com -r user
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

Example:
```
bbx-cli user update -u john -e new_email@example.com
bbx-cli user update -u john -p new_password
```

## Authentication

All user management commands require authentication. The default super admin credentials are loaded from `conf/app.conf`, or you can specify different credentials using the `--host` option.

## Authorization

- Super admin and admin users can register, list, update, and delete any users
- Regular users can only update their own information
- Regular users cannot change their role
- Super admin account cannot be deleted