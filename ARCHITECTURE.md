# Architecture Guide

Deep dive into ZPM's architecture and design patterns.

## System Overview

```
┌────────────────────────────────────────────────────────────┐
│                   User Interface Layer                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ zpm - Command Line Interface                         │  │
│  │ - Argument parsing                                   │  │
│  │ - Command routing                                    │  │
│  │ - User feedback                                      │  │
│  └────────────────┬─────────────────────────────────────┘  │
└─────────────────┼────────────────────────────────────────┬──┘
                  │                                        │
                  │ File I/O                              │ Socket
                  │ Direct Registry Access                │ (future)
                  │                                        │
┌─────────────────▼────────────────────────────────────────▼──┐
│                    Core Layer                                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ registry.zig - Process Registry                      │   │
│  │ - Load/save from disk                                │   │
│  │ - Process lifecycle management                       │   │
│  │ - PID tracking                                       │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ daemon.zig - Process Manager Daemon                  │   │
│  │ - Process spawning (fork/exec)                       │   │
│  │ - Signal handling                                    │   │
│  │ - Command handling                                   │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ System Calls
                     │ - fork()
                     │ - setsid()
                     │ - execve()
                     │ - kill(SIGTERM)
                     │
┌────────────────────▼────────────────────────────────────────┐
│              Operating System Layer                         │
│  - Process Management                                      │
│  - Signal Handling                                         │
│  - File System                                             │
└────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### 1. Main Client (main.zig)

**Purpose**: CLI interface for users

**Responsibilities**:
- Parse command-line arguments
- Route commands to appropriate handlers
- Load registry from disk
- Provide user feedback

**Key Functions**:
- `main()` - Entry point, argument parsing
- Command routing (if/else chain)

**Dependencies**:
- registry.zig
- handlers/* (for legacy mode)

### 2. Registry (registry.zig)

**Purpose**: Single source of truth for process state

**Data Structure**:
```zig
pub const Process = struct {
    name: []const u8,      // Process identifier
    command: []const u8,   // Full command to execute
    status: Status,        // running | stopped
    pid: u32,             // Process ID
};

pub const Status = enum {
    running,
    stopped,
};
```

**Key Functions**:

| Function | Purpose |
|----------|---------|
| `load(allocator)` | Read processes from zpm.json |
| `save()` | Persist to disk |
| `add(name, cmd, pid)` | Register new process |
| `remove(name)` | Mark as stopped |
| `purge(name)` | Delete from registry |
| `stopProcess(name)` | Send SIGTERM |
| `getAll()` | Get all processes |

**Memory Management**:
- Uses GeneralPurposeAllocator
- Strings duplicated when loading to avoid dangling pointers
- Deferred cleanup in all functions

### 3. Daemon (daemon.zig)

**Purpose**: Background process manager

**Responsibilities**:
- Spawn child processes
- Handle signals
- Execute commands from clients
- Maintain daemon state

**Key Functions**:

| Function | Purpose |
|----------|---------|
| `spawnProcess(allocator, cmd)` | Fork + exec |
| `handleCommand(cmd, writer)` | Process user commands |
| `runServer(allocator)` | Main event loop |
| `main()` | Entry point |

**Process Spawning Flow**:
```
spawnProcess()
├─ Parse command into args
├─ fork() → parent & child
│  ├─ Parent: return PID
│  └─ Child:
│     ├─ setsid() - new session
│     ├─ execveZ() - replace image
│     └─ [never returns]
└─ Register in registry
```

### 4. Command Handlers (handlers/)

**Legacy Structure** (for direct registry mode):

- `start.zig` - Handle start command
- `stop.zig` - Handle stop command
- `list.zig` - Handle list command
- `purge.zig` - Handle purge command

## Data Flow

### Starting a Process

```
User Input: zpm start --name app "node server.js"
    ↓
main.zig: Parse args
    ↓
handlers/start.zig: Extract name & command
    ↓
registry.zig:
    ├─ spawn child process via fork/exec
    ├─ get PID from child
    ├─ add() to registry
    └─ save() to disk
    ↓
zpm.json: Updated with new entry
```

### Listing Processes

```
User Input: zpm list
    ↓
main.zig: Recognize list command
    ↓
handlers/list.zig: Query registry
    ↓
registry.zig: getAll() returns array
    ↓
handlers/list.zig: Format and print
    ↓
Terminal: Display process table
```

### Stopping a Process

```
User Input: zpm stop myapp
    ↓
main.zig: Parse stop command
    ↓
handlers/stop.zig: Extract name
    ↓
registry.zig:
    ├─ stopProcess(): kill(pid, SIGTERM)
    ├─ remove(): mark as stopped
    └─ save() to disk
    ↓
zpm.json: Updated status
```

## Execution Modes

### Mode 1: Direct Registry Access (Current)

- Client directly reads/writes zpm.json
- No daemon communication
- Simple but limited

```
zpm → registry.zig → zpm.json
```

### Mode 2: Daemon Communication (Planned)

- Client connects to daemon via Unix socket
- Daemon handles all registry operations
- Better isolation and monitoring

```
zpm ←→ (Unix Socket) ←→ zpmd ↔ registry.zig → zpm.json
```

## Process Lifecycle

```
┌─────────────────────────────────────────────┐
│         Process Lifecycle States            │
└─────────────────────────────────────────────┘

START
  │
  ├─ zpm start --name app "command"
  │
  ▼
REGISTER
  │
  ├─ registry.add(name, cmd, pid)
  ├─ registry.save()
  │
  ▼
RUNNING
  │
  ├─ Child process executing
  ├─ status = "running"
  │
  ├─ Either:
  │  ├─ Natural exit → zombie
  │  └─ zpm stop app
  │
  ▼
STOPPED
  │
  ├─ registry.stopProcess() sends SIGTERM
  ├─ registry.remove() updates status
  ├─ registry.save()
  │
  ├─ Either:
  │  ├─ Keep in registry (restart manually)
  │  └─ zpm purge app
  │
  ▼
PURGED
  │
  ├─ Removed from registry entirely
  ├─ registry.save()
  │
  ▼
END
```

## Memory Management

### Allocation Strategy

```zig
var gpa = std.heap.GeneralPurposeAllocator(.{}){};
defer _ = gpa.deinit();
const allocator = gpa.allocator();
```

### String Ownership

**Strings from JSON parsing**:
```zig
// Strings from parser are freed when parsed.deinit()
const parsed = try std.json.parseFromSlice(...);
defer parsed.deinit();

// Must duplicate before use
for (parsed.value) |p| {
    const name = try allocator.dupe(u8, p.name);
    // name lives as long as allocator
}
```

**Deferred Cleanup**:
```zig
const data = try allocator.alloc(u8, 1024 * 1024);
defer allocator.free(data);
// Automatically freed when scope exits
```

## Error Handling

### Zig Error Model

```zig
fn mayFail() !u32 {
    return error.SomethingWentWrong;
}

pub fn main() !void {
    const result = mayFail() catch |err| {
        std.debug.print("Error: {}\n", .{err});
        return;
    };
}
```

### Registry Errors

```zig
pub fn load(allocator: std.mem.Allocator) !void {
    const file = try std.fs.cwd().openFile(DATA_FILE, .{}) catch return;
    defer file.close();
    
    const data = try file.readToEndAlloc(allocator, 1024 * 1024);
    defer allocator.free(data);
    
    const parsed = try std.json.parseFromSlice(
        []Process,
        allocator,
        data,
        .{},
    );
    defer parsed.deinit();
}
```

## Performance Considerations

### Process Spawning

- **fork()**: ~1-2ms
- **setsid()**: <1ms
- **execveZ()**: ~5-10ms

**Total spawn time**: ~10-15ms

### Registry Operations

- **Load from disk**: <10ms
- **Save to disk**: <5ms
- **In-memory lookup**: <1ms

### Optimization Tips

1. **Batch operations** - Multiple starts in sequence
2. **Lazy loading** - Load registry only when needed
3. **Caching** - Keep registry in memory
4. **Async I/O** - Future daemon implementation

## Concurrency Model

Currently: **Single-threaded, blocking**

- Commands execute sequentially
- File I/O blocks
- Socket operations block (when implemented)

Future: **Event-driven (async)**

- Non-blocking I/O
- Concurrent client handling
- Timer events for monitoring

## Testing Architecture

### Unit Testing

Test individual functions:
```zig
test "registry add" {
    // Test adding process
}
```

### Integration Testing

Test with Hono server:
```bash
zpm start --name test "bun server.ts"
# Verify it's running
curl http://localhost:3000
zpm stop test
```

### Manual Testing

```bash
# Start daemon
zpmd &

# Run commands
zpm start --name app1 "sleep 60"
zpm start --name app2 "sleep 60"
zpm list
zpm stop app1
zpm list
zpm purge app1
zpm list
```

## Security Model

### Process Isolation

- Each process runs independently
- Child processes inherit env from parent
- No built-in sandboxing (run untrusted code with care)

### File Permissions

- `zpm.json` readable/writable by user
- No permission checking for commands
- Registry file world-accessible (single-user only)

### Signal Handling

- SIGTERM for graceful stop (can be caught)
- No SIGKILL (allows cleanup)
- No signal forwarding to children (future)

## Extensibility

### Adding New Commands

1. Create handler in `src/handlers/newcmd.zig`
2. Add case to main.zig routing
3. Test with integration tests

Example:
```zig
// src/handlers/restart.zig
pub fn handleRestart(args: *std.process.ArgIterator) void {
    // Implementation
}
```

### Custom Registry Storage

Replace JSON with:
- SQLite database
- Binary format
- Remote storage

Just swap `registry.zig` implementation.

## Future Enhancements

1. **Distributed Registry** - Multiple machines
2. **Process Groups** - Manage related processes
3. **Resource Limits** - CPU, memory constraints
4. **Auto-restart** - Crash recovery
5. **Metrics** - Performance monitoring
6. **Events** - Webhooks on state changes

---

**Questions?** Open an issue on GitHub.
