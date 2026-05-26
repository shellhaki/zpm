# ZPM

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Linux](https://img.shields.io/badge/Linux-ready-FCC624?style=for-the-badge&logo=linux&logoColor=000)](#)
[![macOS](https://img.shields.io/badge/macOS-ready-000000?style=for-the-badge&logo=apple)](#)
[![Windows](https://img.shields.io/badge/Windows-ready-0078D4?style=for-the-badge&logo=windows)](#)

**ZPM is a small, sharp process manager for apps you want to keep alive.**

It gives you a daemon, named processes, crash restart, log following, clusters, environment profiles, health checks, log rotation, startup on boot, and ecosystem config files without turning your terminal into a carnival.

## Quick Install

### One-Liner Installation

**Linux/macOS (Bash):**
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash
```

**Windows (PowerShell as Administrator):**
```powershell
irm https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install-windows.ps1 | iex
```

### What Gets Installed

- **zpm** - Command-line interface for managing processes
- **zpmd** - Background daemon (auto-starts on boot)
- **PATH** - Binary location automatically added to your system PATH

### After Installation

```bash
zpm --help              # See available commands
zpmd                    # Start daemon (or auto-starts on boot)
zpm list                # List managed processes
zpm tui                 # Open the animated terminal dashboard
zpm start script.js     # Start managing a process
```

### Supported Platforms

✅ **Linux**: amd64, arm64 (auto-detected)  
✅ **macOS**: amd64 (Intel), arm64 (Apple Silicon) (auto-detected)  
✅ **Windows**: amd64

### Daemon Auto-Start

| Platform | Managed By | Enable Command |
|----------|-----------|-----------------|
| **Linux** | systemd user service | `systemctl --user enable zpmd` |
| **macOS** | LaunchAgent | `launchctl load ~/Library/LaunchAgents/com.zpm.daemon.plist` |
| **Windows** | Task Scheduler | Auto-configured if installer run as Admin |

## Install From Source

```bash
go build -o daemon/zpmd ./daemon
go build -o src/zpm ./src
ln -sf "$PWD/src/zpm" ~/.local/bin/zpp
```

Keep `zpmd` beside the CLI or set:

```bash
export ZPMD_PATH=/path/to/zpmd
```

## Daemon

```bash
zpp daemon start
zpp daemon stop
zpp daemon reload
```

Start on login:

```bash
zpp startup install
zpp startup uninstall
```

Linux uses user systemd, macOS uses LaunchAgent, Windows uses Task Scheduler.

## Run Apps

```bash
zpp start "bun index" --name api --follow
zpp status
zpp tui
zpp stop api
zpp restart api
zpp start api
zpp purge api
```

ZPM starts commands from the directory where you run `zpp start`, so package scripts work naturally.

```json
{
  "scripts": {
    "serve:zpm": "zpp start \"bun index\" --name api --follow"
  }
}
```

## PM2-Style Features

```bash
zpp start "bun index" --name api --env production
zpp start "bun index" --name api --env PORT=3000
zpp start "bun index" --name api --instances 4
zpp start "bun index" --name api --restart-delay 1000 --max-restarts 10
zpp start "bun index" --name api --health "curl -fsS http://127.0.0.1:3000/health"
zpp start "bun index" --name api --log-max-size 20mb --log-backups 7
```

Cluster instances are named `api-0`, `api-1`, etc. Group commands work:

```bash
zpp stop api
zpp restart api
zpp purge api
```

## Terminal Dashboard

```bash
zpm tui
zpm ui
zpm tux
zpm status --watch
```

The dashboard refreshes live, animates daemon activity, adapts to narrow and wide terminals, and lets you move through processes with `j/k` or arrow keys.

## Ecosystem Config

Create `zpm.config.json`:

```json
{
  "apps": [
    {
      "name": "api",
      "command": "bun index",
      "cwd": ".",
      "instances": 2,
      "auto_restart": true,
      "restart_delay": 1000,
      "max_restarts": -1,
      "health_command": "curl -fsS http://127.0.0.1:3000/health",
      "log_max_bytes": 10485760,
      "log_backups": 5,
      "env": {
        "PORT": "3000"
      },
      "env_production": {
        "NODE_ENV": "production"
      }
    }
  ]
}
```

Run it:

```bash
zpp ecosystem start zpm.config.json --env production
```

## Storage

ZPM stores state in your user config directory:

```text
registry.json
daemon.pid
daemon.log
logs/<process>.log
```

On Linux that is usually `~/.config/zpm`.

## Installation Scripts (Curl-Based)

The one-liner installers are cross-platform shell/PowerShell scripts that automate installation. This section describes how they work and the implementation details.

### How It Works

The universal installer (`install.sh`) detects your operating system and architecture, then routes to the appropriate platform-specific installer:

```
User runs: curl | bash
    ↓
install.sh (universal entry point)
    ↓
    ├─→ Linux? → install-linux.sh
    ├─→ macOS? → install-macos.sh
    └─→ Windows? → install-windows.ps1
```

### Platform-Specific Installers

#### Linux (install-linux.sh)
- **Architecture detection**: Auto-detects amd64 or arm64 via `uname -m`
- **Download**: Fetches `zpm-linux-{arch}.tar.gz` from GitHub Releases
- **Installation**: Extracts to `~/.local/bin/`
- **PATH setup**: Adds export to `~/.bashrc` or `~/.zshrc` (shell auto-detected)
- **Daemon**: Creates systemd user service at `~/.config/systemd/user/zpmd.service`
- **Error handling**: Validates binaries and URLs, comprehensive error messages

#### macOS (install-macos.sh)
- **Architecture detection**: Auto-detects amd64 or arm64 via `uname -m`
- **Download**: Fetches `zpm-darwin-{arch}.tar.gz` from GitHub Releases
- **Installation**: Extracts to `~/.local/bin/`
- **PATH setup**: Adds export to `~/.zshrc` or `~/.bash_profile` (shell auto-detected)
- **Daemon**: Creates LaunchAgent at `~/Library/LaunchAgents/com.zpm.daemon.plist`
- **Logging**: Creates `~/.zpm/` directory for daemon logs
- **Error handling**: Validates binaries and URLs, color-coded output

#### Windows (install-windows.ps1)
- **Architecture detection**: Uses `PROCESSOR_ARCHITECTURE` environment variable
- **Download**: Fetches `zpm-windows-amd64.tar.gz` from GitHub Releases
- **Installation**: Extracts to `%LOCALAPPDATA%\zpm\bin`
- **PATH setup**: Updates User PATH environment variable (persists across sessions)
- **Daemon**: Creates Task Scheduler task "ZPM Daemon" (if run as Administrator)
- **Error handling**: Graceful handling when not run as Administrator

### Key Features

✅ **No external tools** - Uses only curl, tar, and basic shell utilities  
✅ **Auto-detection** - OS, architecture, and shell all auto-detected  
✅ **User-level install** - No sudo/admin required (except Windows Task Scheduler)  
✅ **Platform-native services** - Uses systemd/LaunchAgent/Task Scheduler  
✅ **Error handling** - Validates downloads, verifies binaries, helpful messages  
✅ **Color output** - Clear, friendly terminal feedback  

### GitHub Release Integration

All scripts fetch from GitHub API:
```
https://api.github.com/repos/shellhaki/zpm/releases/latest
or
https://api.github.com/repos/shellhaki/zpm/releases/tags/{version}
```

Binary assets are matched by platform pattern:
- Linux: `zpm-linux-{arch}.tar.gz`
- macOS: `zpm-darwin-{arch}.tar.gz`
- Windows: `zpm-windows-amd64.tar.gz`

### Custom Installation Directory

**Linux/macOS (specify custom directory as second argument):**
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash -s latest /usr/local/bin
```

**Windows PowerShell:**
```powershell
$InstallDir = "C:\Program Files\zpm"
# Download and edit the script with custom InstallDir
```

### Specify Version

**Linux/macOS (specific version as first argument):**
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash -s v1.0.0
```

### Troubleshooting

**Command not found after install?**
- Linux/macOS: Run `source ~/.bashrc` or `source ~/.zshrc`
- Windows: Restart your terminal/PowerShell

**Daemon won't start?**
- Linux: Check with `systemctl --user status zpmd`
- macOS: Check with `launchctl list | grep zpm`
- Windows: Check Task Scheduler for "ZPM Daemon" task

**Failed to download?**
- Verify internet connection
- Check GitHub is accessible
- Try specifying an exact version instead of "latest"

### Uninstall

**Linux:**
```bash
systemctl --user disable zpmd
rm ~/.local/bin/{zpm,zpmd}
# Edit ~/.bashrc or ~/.zshrc to remove ZPM PATH export
```

**macOS:**
```bash
launchctl unload ~/Library/LaunchAgents/com.zpm.daemon.plist
rm -rf ~/.local/bin/{zpm,zpmd} ~/.zpm ~/Library/LaunchAgents/com.zpm.daemon.plist
# Edit ~/.zshrc or ~/.bash_profile to remove ZPM PATH export
```

**Windows (PowerShell as Administrator):**
```powershell
Unregister-ScheduledTask -TaskName "ZPM Daemon" -Confirm:$false
Remove-Item "$env:LOCALAPPDATA\zpm" -Recurse
# Remove from User PATH environment variables manually
```

### Script Files

Located in the `scripts/` directory:
- **install.sh** - Universal entry point (OS detection and routing)
- **install-linux.sh** - Linux-specific installer with systemd integration
- **install-macos.sh** - macOS-specific installer with LaunchAgent integration
- **install-windows.ps1** - Windows PowerShell installer with Task Scheduler integration
- **README.md** - Detailed installation and troubleshooting guide

## Release

Push a tag to publish cross-platform builds:

```bash
git tag v1.0.1
git push origin v1.0.1
```

The release pipeline builds:

- Linux amd64, arm64
- macOS amd64, arm64
- Windows amd64
