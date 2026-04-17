# License Feature Code Design

## Summary
Add a CLI command and REST API endpoint to retrieve machine-bound license/feature codes based on physical network interfaces (MAC addresses). The implementation ports the logic from the Java `SCUtils` class to Go.

## Requirements
- Enumerate all network interfaces on the server.
- Filter out: loopback, interfaces without a hardware address, Docker interfaces (`docker` in name), and common virtual interfaces (`veth`, `virbr`, `dummy`, `tun`, `tap`, `vEthernet`, `Virtual`, `Hyper-V`).
- For each valid interface, pick the first usable IP address (prefer IPv4, skip loopback / multicast / any-local).
- Compute the feature code matching Java `SCUtils` behavior:
  1. Format MAC address as uppercase hex with `-` separators (e.g., `00-1A-2B-3C-4D-5E`).
  2. Concatenate salt `"AAS"` + MAC.
  3. SHA-256 hash the string.
  4. Convert the hash bytes to a signed 32-bit integer (matching `BigInteger(1, hash).intValue()`).
  5. Convert the integer to a decimal string and prefix with `"SZTY"`.
- Return the code together with the interface name and IP address.

## Architecture

### New Files
- `cli/license.go`: CLI command registration and handler.
- `api/handlers_license.go`: HTTP handler for the license endpoint.

### Modified Files
- `api/routes.go`: Register `GET /api/license`.
- `api/models.go`: Add `LicenseInfo` struct with `code` field.
- `api/client_core.go`: Add `GetLicense() ([]LicenseInfo, error)` client method.
- `cli/commands.go`: Register the `license` command under the root command.

## Data Model
```go
type LicenseInfo struct {
    Code      string `json:"code"`
    Interface string `json:"interface"`
    IP        string `json:"ip"`
}
```

## API Endpoint
- `GET /api/license`
- Response: `200 OK` with a JSON array of `LicenseInfo` objects.

## CLI Command
- `bbx-cli license`
- Prints a table with columns: `CODE`, `INTERFACE`, `IP`.

## Error Handling
- If no valid interfaces are found, the API returns an empty array (`[]`) with `200 OK`.
- The CLI prints a friendly message when no codes are available.

## Testing
- Build the server and CLI.
- Run `bbx-cli license` against a running server and verify output includes only physical NICs.
- Verify the generated codes match the Java implementation for the same MAC address.
