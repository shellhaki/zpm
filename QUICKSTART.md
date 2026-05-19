# Quick Start Guide

Get up and running with ZPM in 5 minutes.

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/zpm.git
cd zpm

# Build the project
zig build
```

**Output:**
```
Install Summary:
- install zpm
- install zpmd
```

Binaries are now in `zig-out/bin/`.

## First Steps

### 1. Navigate to the bin directory

```bash
cd zig-out/bin
```

### 2. Start a simple command

```bash
./zpm start --name hello "sleep 300"
```

**Output:**
```
process started
name: hello
command: sleep 300
```

### 3. List running processes

```bash
./zpm list
```

**Output:**
```
hello  [running]  sleep 300
```

### 4. Stop the process

```bash
./zpm stop hello
```

### 5. Remove from registry

```bash
./zpm purge hello
```

## Next: Test with Real Server

### Install Dependencies

```bash
# Install Bun if you don't have it
curl -fsSL https://bun.sh/install | bash

# Install test server dependencies
cd ../test-server
bun install
```

### Run the Test Server

```bash
cd ../../zig-out/bin

# Start Hono server with ZPM
./zpm start --name server "bun /home/haki/zpm/test-server/src/index.ts"

# Verify it's running
./zpm list

# In another terminal, test it
curl http://localhost:3000/health
curl http://localhost:3000/api/info

# Stop it
./zpm stop server
./zpm purge server
```

## Common Commands

| Command | Purpose |
|---------|---------|
| `zpm start --name APP "CMD"` | Start process named APP running CMD |
| `zpm list` | Show all processes |
| `zpm stop NAME` | Stop process by name |
| `zpm purge NAME` | Remove process from registry |

## Project Files

After building, you'll have:

```
zpm/
├── zig-out/
│   └── bin/
│       ├── zpm       ← Client CLI
│       ├── zpmd      ← Daemon (currently demo mode)
│       └── zpm.json  ← Process registry
├── src/              ← Source code
├── test-server/      ← Bun+Hono test app
├── README.md         ← Full documentation
└── CONTRIBUTING.md   ← How to contribute
```

## Troubleshooting

### Command not found

Make sure you're in the right directory:
```bash
cd /path/to/zpm/zig-out/bin
```

### Port already in use

If port 3000 is in use:
```bash
# Find what's using it
lsof -i :3000

# Kill it
kill -9 <PID>
```

### Build fails

Make sure you have Zig 0.12.0+:
```bash
zig version
```

## Next Steps

1. **Read** the [README.md](README.md) for detailed documentation
2. **Check** [CONTRIBUTING.md](CONTRIBUTING.md) to contribute
3. **Explore** the source code in `src/`
4. **Join** discussions for questions

## Resources

- [ZPM README](README.md)
- [Contributing Guide](CONTRIBUTING.md)
- [Zig Language](https://ziglang.org)
- [Hono Framework](https://hono.dev)
- [Bun Runtime](https://bun.sh)

## Getting Help

- **Issues** - Report bugs or request features
- **Discussions** - Ask questions and share ideas
- **Documentation** - Check README and guides

---

**Happy process managing! 🚀**
