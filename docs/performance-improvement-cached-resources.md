# Performance Improvement: Cached Resource Monitoring

## Overview
The `setupResourceCommands` in `cli/commands.go` has been optimized by implementing a scheduled collection and caching mechanism. Previously, the command would call expensive real-time collection functions every time, causing delays due to mandatory 1-second sleeps for dual sampling of network and disk I/O rates.

## Implementation Details

### 1. Resource Cache (`proc/resource_monitor.go`)
- Added `ResourceCache` struct with thread-safe operations
- Implemented TTL (time-to-live) for cached data to ensure freshness
- Separate caching for system resources and individual process resources
- Methods for getting/setting cached resource information

### 2. Resource Monitor Service (`proc/resource_monitor.go`)
- Created `ResourceMonitor` that runs as a background routine
- Collects system and process metrics at configurable intervals (default: every 3 seconds)
- Updates cache with fresh data in the background
- Provides cached data via `GetCachedSystemResources()` and `GetCachedProcessResources()` methods

### 3. API Handler Updates (`api/handlers.go`)
- Updated `handleGetSystemResources()` to first check cache before falling back to real-time collection
- Updated `handleGetProcessResources()` to use cached data when available
- Maintains backward compatibility for real-time collection when cache is unavailable

### 4. Server Integration (`api/server.go`)
- Integrated `ResourceMonitor` into the API server
- Initialized with appropriate TTL (5 seconds) and collection interval (3 seconds)
- Added graceful shutdown support in `Stop()` method

## Performance Benefits

- **Reduced Latency**: API responses now return cached data in milliseconds instead of waiting for 1+ second collection
- **Consistent Performance**: No more variable response times due to real-time collection
- **Efficient Resource Usage**: Background collection prevents redundant computation
- **Configurable Freshness**: Cache TTL ensures data remains reasonably current

## Configuration Parameters

- Cache TTL: 5 seconds (data older than this is considered stale)
- Collection Interval: 3 seconds (frequency of background collection)
- Toggle options for collecting system/process data independently

## Files Modified

1. `proc/resource_monitor.go` - New file containing cache and monitor implementation
2. `proc/resource_monitor_test.go` - Unit tests for the new functionality
3. `api/server.go` - Integrated resource monitor into server lifecycle
4. `api/handlers.go` - Updated handlers to use cached data
5. `proc/models.go` - Already contained ResourceStats structure (dependency)

## Backward Compatibility

- Existing functions remain unchanged for other use cases requiring real-time data
- CLI commands (`bbx-cli resources system/process`) will now respond much faster
- API endpoints maintain the same interface and response format
- No breaking changes to existing functionality