# Installation & Build Guide

Complete instructions for building ZPM from source.

## Requirements

### Minimum

- **Zig** 0.12.0 or later
- **Linux/UNIX** system (macOS, Linux)
- **C compiler** (usually comes with Zig)

### Optional

- **Bun** 1.0+ (for test server)
- **Git** (for cloning)
- **Make** or **Just** (for build automation)

## Installing Zig

### Linux

```bash
# Download latest Zig
wget https://ziglang.org/download/0.12.0/zig-linux-x86_64-0.12.0.tar.xz

# Extract
tar xf zig-linux-x86_64-0.12.0.tar.xz

# Add to PATH
export PATH=$PWD/zig-0.12.0:$PATH
```

### macOS

```bash
# Using Homebrew
brew install zig

# Or download directly
curl -O https://ziglang.org/download/0.12.0/zig-macos-aarch64-0.12.0.tar.xz
tar xf zig-macos-aarch64-0.12.0.tar.xz
export PATH=$PWD/zig-0.12.0/bin:$PATH
```

### Verify Installation

```bash
zig version
# Output: 0.12.0
```

## Building ZPM

### 1. Clone Repository

```bash
git clone https://github.com/shellhaki/zpm.git
cd zpm
```

### 2. Build

#### Debug Build

```bash
zig build
```

#### Release Build (Optimized)

```bash
zig build -Doptimize=ReleaseSafe
```

#### Release with Full Optimization

```bash
zig build -Doptimize=ReleaseSmall
```

#### Build Output

```
install
├─ install zpm          → Client CLI
└─ install zpmd         → Daemon
Build Summary: 2/2 steps succeeded
```

Binaries are in `zig-out/bin/`:

- `zpm` - Process manager client (~2.8MB)
- `zpmd` - Daemon server (~2.7MB)

### 3. Verify Build

```bash
cd zig-out/bin

# Check if binaries exist
ls -lh zpm zpmd

# Test client
./zpm
# Output: Invalid usage

# Test daemon
./zpmd
# Output: zpmd daemon starting...
#         zpmd daemon started
```

Press `Ctrl+C` to stop daemon.

## Development Build

### Incremental Building

Zig caches builds automatically. Rebuilds are fast:

```bash
zig build      # ~2-3s for incremental changes
```

### Clean Build

```bash
rm -rf zig-cache zig-out
zig build
```

### Build Specific Binary

```bash
# Build only client
zig build-exe -Mroot=src/main.zig -o zpm

# Build only daemon
zig build-exe -Mroot=src/daemon.zig -o zpmd
```

## Installing Globally

### Option 1: Add to PATH

```bash
# Add to .bashrc or .zshrc
export PATH=$HOME/zpm/zig-out/bin:$PATH

# Reload shell
source ~/.bashrc
```

### Option 2: Symlink

```bash
sudo ln -s ~/zpm/zig-out/bin/zpm /usr/local/bin/zpm
sudo ln -s ~/zpm/zig-out/bin/zpmd /usr/local/bin/zpmd

# Verify
which zpm
which zpmd
```

### Option 3: Copy to System

```bash
sudo cp zig-out/bin/zpm /usr/local/bin/
sudo cp zig-out/bin/zpmd /usr/local/bin/

# Make executable
sudo chmod +x /usr/local/bin/zpm
sudo chmod +x /usr/local/bin/zpmd
```

## Building Test Server

### Install Bun

```bash
curl -fsSL https://bun.sh/install | bash

# Add to PATH if not automatic
export PATH=$HOME/.bun/bin:$PATH
```

### Install Dependencies

```bash
cd test-server
bun install
```

### Run Test Server

```bash
bun run src/index.ts
```

## Troubleshooting Build Issues

### "command not found: zig"

Zig is not in your PATH. See "Installing Zig" section above.

### "error: unable to create output directory"

```bash
# Check permissions
ls -la zig-out/

# Try creating directory manually
mkdir -p zig-out/bin
zig build
```

### "error: libc.zig: no such file or directory"

Zig installation is incomplete. Reinstall Zig.

### "error: C compiler not found"

Install a C compiler:

```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# CentOS/RHEL
sudo yum groupinstall "Development Tools"

# macOS
xcode-select --install
```

### Slow builds

1. Check available disk space
2. Use Release build: `zig build -Doptimize=ReleaseSafe`
3. Close other programs

## Cross-Compilation

Build for other architectures:

```bash
# For ARM64 (macOS Apple Silicon simulation)
zig build-exe src/main.zig -target aarch64-linux

# For 32-bit Linux
zig build-exe src/main.zig -target i386-linux

# For Windows (experimental)
zig build-exe src/main.zig -target x86_64-windows
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: goto-bus-stop/setup-zig@v2
      - run: zig build
      - run: ./zig-out/bin/zpm --help
```

### Local Testing

```bash
#!/bin/bash
set -e

echo "Building..."
zig build

echo "Running tests..."
cd zig-out/bin
./zpm start --name test "sleep 10"
./zpm list
./zpm stop test
./zpm purge test

echo "Build successful!"
```

## Docker Build

### Dockerfile

```dockerfile
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    curl \
    build-essential

# Install Zig
RUN curl https://ziglang.org/download/0.12.0/zig-linux-x86_64-0.12.0.tar.xz | tar xJ
ENV PATH="/zig-0.12.0:$PATH"

WORKDIR /zpm
COPY . .

RUN zig build

CMD ["./zig-out/bin/zpmd"]
```

### Build Docker Image

```bash
docker build -t zpm:latest .
docker run -it zpm:latest
```

## Size Optimization

### Reduce Binary Size

```bash
# Release with size optimization
zig build -Doptimize=ReleaseSmall

# Strip symbols
strip zig-out/bin/zpm

# Compression
upx zig-out/bin/zpm
```

Binary sizes:
- Debug: ~2.8MB
- Release: ~2.4MB
- ReleaseSmall: ~2.0MB
- Stripped + compressed: ~700KB

## Performance Tuning

### Build Performance

```bash
# Parallel compilation
zig build -j 4

# Cache clearing (if build is stuck)
rm -rf zig-cache
zig build
```

## Uninstalling

### Remove from PATH

```bash
# Edit ~/.bashrc or ~/.zshrc and remove the PATH line
nano ~/.bashrc

# Reload
source ~/.bashrc
```

### Remove Symlinks

```bash
sudo rm /usr/local/bin/zpm
sudo rm /usr/local/bin/zpmd
```

### Remove Source

```bash
rm -rf ~/zpm
```

## Next Steps

1. Read the [Quick Start Guide](QUICKSTART.md)
2. Check the [README](README.md)
3. Review [Contributing Guide](CONTRIBUTING.md)

---

**For more help**, open an issue on GitHub.
