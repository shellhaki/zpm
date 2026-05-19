# ZPM - Zig Process Manager

A lightweight, functional process manager written in Zig. Similar to PM2, ZPM spawns and manages background processes with a clean CLI interface.

## Features

- **Process Spawning** - Fork and detach processes from the terminal
- **Process Tracking** - Maintain registry of running/stopped processes with PIDs
- **Signal Management** - Gracefully stop processes with SIGTERM
- **Process Purging** - Remove processes from the registry
- **Persistent Registry** - JSON-based process state storage
- **Daemon Architecture** - Background daemon for monitoring
- **Pure Zig** - No external dependencies, functional programming paradigm
- **Unix Sockets** - IPC communication between client and daemon

## Architecture

```
┌─────────────────────────────────────────────────────┐
│ zpm (client)                                        │
│ - CLI argument parsing                              │
│ - User-facing commands (start/stop/list/purge)     │
│ - Socket communication to daemon                    │
└──────────────────┬──────────────────────────────────┘
                   │
                   │ Unix Socket (/tmp/zpmd.sock)
                   │
┌──────────────────▼──────────────────────────────────┐
│ zpmd (daemon)                                       │
│ - Process spawning (fork/setsid)                   │
│ - Signal handling (SIGTERM for graceful stop)      │
│ - Registry management                              │
│ - Process monitoring                               │
└──────────────────┬──────────────────────────────────┘
                   │
                   │ File I/O
                   │
         ┌─────────▼──────────┐
         │ zpm.json           │
         │ (Process Registry) │
         └────────────────────┘
```

## Installation

### Prerequisites

- Zig 0.12.0 or later
- Linux/UNIX system
- Bun (optional, for test server)

### Build from Source

```bash
git clone https://github.com/yourusername/zpm.git
cd zpm
zig build
```

Binaries will be in `zig-out/bin/`:
- `zpm` - Client CLI
- `zpmd` - Daemon process

## Usage

### Starting the Daemon

```bash
./zpmd
# Output: zpmd daemon starting...
# Output: zpmd daemon started
```

The daemon runs in the foreground. Use `&` to background it or run in screen/tmux:

```bash
zpmd &
# or
screen -dmS zpmd zpmd
```

### Commands

#### Start a Process

```bash
zpm start --name myapp "node app.js"
# Output: process started
#         name: myapp
#         command: node app.js
```

#### List Processes

```bash
zpm list
# Output: myapp  [running]  node app.js
#         other  [stopped]  redis-server
```

#### Stop a Process

```bash
zpm stop myapp
# Sends SIGTERM to the process, marks as stopped
```

#### Remove from Registry

```bash
zpm purge myapp
# Completely removes process from registry
```

## Data Format

Processes are stored in `zpm.json` (in the current directory):

```json
[
  {
    "name": "web_server",
    "command": "bun start",
    "status": "running",
    "pid": 12345
  },
  {
    "name": "background_job",
    "command": "node worker.js",
    "status": "stopped",
    "pid": 0
  }
]
```

## Example: Testing with Hono Server

```bash
cd test-server
bun install
cd ../zig-out/bin

./zpm start --name hono-test "bun /path/to/test-server/src/index.ts"
./zpm list
curl http://localhost:3000/health
./zpm stop hono-test
```

## Project Structure

```
zpm/
├── src/
│   ├── main.zig           # Client CLI
│   ├── daemon.zig         # Daemon process manager
│   ├── registry.zig       # Process registry & persistence
│   ├── handlers/          # Command handlers
│   │   ├── start.zig
│   │   ├── stop.zig
│   │   ├── list.zig
│   │   └── purge.zig
│   └── build.zig          # Build configuration
├── test-server/           # Bun+Hono test application
│   ├── package.json
│   ├── src/index.ts
│   └── README.md
└── README.md              # This file
```

## Development

### Building

```bash
zig build          # Debug build
zig build -Doptimize=ReleaseSafe  # Release build
```

### Key Modules

**registry.zig** - Core process registry
- `load(allocator)` - Load processes from disk
- `save()` - Persist to disk
- `add(name, command, pid)` - Register new process
- `remove(name)` - Mark as stopped
- `purge(name)` - Delete from registry
- `stopProcess(name)` - Send SIGTERM

**daemon.zig** - Daemon server
- `spawnProcess(command)` - Fork and exec
- `handleCommand(cmd, writer)` - Process command
- `main()` - Entry point

**main.zig** - Client interface
- Argument parsing
- Command routing
- Registry operations

## Design Philosophy

ZPM follows functional programming principles:

- **No mutability** - State changes are explicit
- **No global state** - All data passed through function parameters
- **Composable functions** - Small, focused functions
- **No comments** - Code clarity through naming
- **Error handling** - Explicit with try/catch

## Limitations & Future Work

- [ ] Socket-based daemon communication (currently direct registry access)
- [ ] Process restart on crash
- [ ] Resource limits (memory/CPU)
- [ ] Process groups
- [ ] Multi-user support
- [ ] Logs aggregation
- [ ] Web dashboard
- [ ] Cross-platform support (Windows)

## Performance

- Startup: <50ms
- Process spawn: ~100-200ms
- Registry load/save: <10ms

## Troubleshooting

### Daemon won't start

```bash
# Check if port is in use
lsof -i :3000

# Check logs
zpmd 2>&1
```

### Process not stopping

```bash
# Manual process kill
ps aux | grep "your_process"
kill -SIGTERM <pid>
```

### Registry corruption

```bash
# Backup and reset
mv zpm.json zpm.json.bak
# Restart zpmd
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- Follow Zig conventions
- Use snake_case for variables
- Keep functions small and pure
- Add error handling with try/catch
- No code comments - use clear naming

## License

MIT License - see LICENSE file for details

## Inspiration

- PM2 - Production process manager for Node.js
- Supervisor - Process control system
- systemd - System and service manager

## Author

Built with Zig, for learning and production use.

---

**Status**: Early development - API may change

**Last Updated**: May 2026
