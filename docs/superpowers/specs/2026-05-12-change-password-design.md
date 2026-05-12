# Super Admin Password Change API Design

## Overview

Add a dedicated API endpoint that allows the super admin (defined in `conf/config.yaml`) to change their own password. The new password is persisted back to the config YAML file and updated in-memory.

## Motivation

Currently there is no way for the super admin to change the password stored in `conf/config.yaml` without manually editing the file. The existing `PUT /api/config` endpoint replaces the entire config, which is unsafe for this purpose (exposes sensitive fields like `encryption_key` and does not verify the current password).

## API Design

### Endpoint

```
POST /api/auth/change-password
```

### Request Body

```json
{
  "current_password": "Flzx3qL@ysyhl9t",
  "new_password": "NewP@ssw0rd123"
}
```

### Response

**Success (200)**
```json
{
  "message": "Password changed successfully"
}
```

**Errors**
- `400` — Invalid request body or missing fields
- `401` — No Basic Auth provided
- `403` — Current password verification failed, or caller is not the super admin
- `500` — Failed to write config file

## Authentication & Authorization

- The endpoint requires Basic Auth to be enabled (`s.config.BasicAuth.Enabled`)
- Only the super admin (username matching `s.config.BasicAuth.Username`) can call this endpoint
- The `current_password` must match `s.config.BasicAuth.Password` before the change is applied

## Implementation Details

### Server Struct Change

Add `configPath string` to `api.Server` so the server knows where to persist config changes:

```go
type Server struct {
    config     *config.Config
    configPath string   // NEW: path to the config YAML file
    // ... rest unchanged
}
```

### Constructor Change

`NewServerWithManager` accepts an additional `configPath string` parameter:

```go
func NewServerWithManager(config *config.Config, manager *proc.Manager, configPath string) *Server
```

### Handler Logic (`handleChangePassword`)

1. Verify Basic Auth is present and the username matches `s.config.BasicAuth.Username`
2. Decode request body `{ current_password, new_password }`
3. Validate both fields are non-empty
4. Compare `current_password` with `s.config.BasicAuth.Password`
5. If match: update `s.config.BasicAuth.Password = new_password`
6. Call `config.SaveConfig(s.configPath, s.config)` to persist
7. Return success

### Refactor: Remove Hardcoded Config Path

The existing `handleUpdateConfig` currently uses hardcoded `"./conf/config.yaml"`. Update it to use `s.configPath`.

## Files to Modify

| File | Change |
|------|--------|
| `api/server.go` | Add `configPath` field to `Server`; update `NewServerWithManager` signature |
| `api/handlers_core.go` | Add `handleChangePassword` handler; update `handleUpdateConfig` to use `s.configPath` |
| `api/routes.go` | Register `POST /api/auth/change-password` route |
| `cmd/bbx-server/main.go` | Pass `configPath` to `NewServerWithManager` |

## Out of Scope

- Password strength validation (min length, complexity) — not required
- Support for registered users (SQLite-backed) changing password — they already have `PUT /api/users/{username}`
- Session/token invalidation after password change — Basic Auth is stateless
