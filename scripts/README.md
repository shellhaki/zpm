# ZPM Installation Scripts

One-line installers for ZPM (Zen Process Manager) across Linux, macOS, and Windows.

## Quick Start

### Linux & macOS
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash
```

Or with a specific version:
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash -s v1.0.0
```

### Windows (PowerShell)
Run **PowerShell as Administrator**, then:
```powershell
irm https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install-windows.ps1 | iex
```

Or with a specific version:
```powershell
$script = irm https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install-windows.ps1
$block = [scriptblock]::Create($script)
& $block -Version "v1.0.0"
```

## What Gets Installed

- **zpm**: Command-line interface for managing processes
- **zpmd**: Background daemon that manages processes
- Automatic PATH configuration
- Platform-specific daemon setup (systemd, LaunchAgent, or Task Scheduler)

### Installation Locations

| Platform | Location |
|----------|----------|
| Linux | `~/.local/bin/` |
| macOS | `~/.local/bin/` |
| Windows | `%LOCALAPPDATA%\zpm\bin` |

## Installation Details

### Linux
- Downloads architecture-appropriate binary (amd64/arm64)
- Extracts to `~/.local/bin/`
- Adds to PATH in `~/.bashrc` or `~/.zshrc`
- Creates systemd user service: `~/.config/systemd/user/zpmd.service`
- **To enable zpmd on startup:**
  ```bash
  systemctl --user enable zpmd
  systemctl --user start zpmd
  ```

### macOS
- Downloads architecture-appropriate binary (amd64/arm64)
- Extracts to `~/.local/bin/`
- Adds to PATH in `~/.zshrc` or `~/.bash_profile`
- Creates LaunchAgent: `~/Library/LaunchAgents/com.zpm.daemon.plist`
- **To enable zpmd on startup:**
  ```bash
  launchctl load ~/Library/LaunchAgents/com.zpm.daemon.plist
  ```
- Logs stored in `~/.zpm/`

### Windows
- Downloads amd64 binary
- Extracts to `%LOCALAPPDATA%\zpm\bin`
- Adds to User PATH environment variable
- Creates Task Scheduler task: "ZPM Daemon" (if run as Administrator)
- Task runs zpmd automatically on logon
- **Manual startup:**
  ```cmd
  zpmd
  ```

## Custom Installation Directory

### Linux & macOS
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash -s latest /usr/local/bin
```

### Windows
```powershell
$script = irm https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install-windows.ps1
$block = [scriptblock]::Create($script)
& $block -Version "latest" -InstallDir "C:\Program Files\zpm"
```

## Manual Installation (Build from Source)

```bash
# Clone repository
git clone https://github.com/shellhaki/zpm
cd zpm

# Build daemon
cd daemon
go build -o ../zpmd .

# Build CLI
cd ../src
go build -o ../zpm .

# Install to ~/.local/bin
mkdir -p ~/.local/bin
cp ../zpm ~/.local/bin/
cp ../zpmd ~/.local/bin/
chmod +x ~/.local/bin/{zpm,zpmd}
```

## Verify Installation

After installation:
```bash
# Check zpm
zpm --help

# Start daemon
zpmd

# In another terminal, verify daemon is running
zpm list
```

## Troubleshooting

### Command not found: zpm/zpmd
- **Linux/macOS**: Make sure to run `source ~/.bashrc` or `source ~/.zshrc`
- **Windows**: Restart your terminal

### Permission denied
- **Linux/macOS**: Check that `~/.local/bin/` is in your PATH and files are executable
- **Windows**: Run PowerShell as Administrator

### Failed to download
- Verify internet connection
- Check that GitHub is accessible
- Specify an exact version instead of "latest"

### zpmd won't start
- **Linux**: Check `systemctl --user status zpmd`
- **macOS**: Check `launchctl list | grep zpm`
- **Windows**: Check Task Scheduler for "ZPM Daemon"

## Uninstall

### Linux
```bash
# Disable systemd service
systemctl --user disable zpmd
systemctl --user stop zpmd

# Remove binaries
rm ~/.local/bin/zpm ~/.local/bin/zpmd

# Remove PATH from shell config (edit ~/.bashrc or ~/.zshrc manually)
```

### macOS
```bash
# Unload LaunchAgent
launchctl unload ~/Library/LaunchAgents/com.zpm.daemon.plist

# Remove files
rm ~/Library/LaunchAgents/com.zpm.daemon.plist
rm ~/.local/bin/zpm ~/.local/bin/zpmd
rm -rf ~/.zpm

# Remove PATH from shell config (edit ~/.zshrc or ~/.bash_profile manually)
```

### Windows
```powershell
# Delete Task Scheduler task (as Administrator)
Unregister-ScheduledTask -TaskName "ZPM Daemon" -Confirm:$false

# Remove binaries
Remove-Item "$env:LOCALAPPDATA\zpm" -Recurse

# Remove from PATH (as Administrator):
# Settings > System > About > Environment variables > Edit User environment variables
# Remove %LOCALAPPDATA%\zpm\bin from PATH
```

## Scripts Overview

- **install.sh**: Main entry point that detects OS and runs appropriate installer
- **install-linux.sh**: Linux-specific installer (auto-detects amd64/arm64)
- **install-macos.sh**: macOS-specific installer (auto-detects amd64/arm64)  
- **install-windows.ps1**: Windows PowerShell installer (amd64 only)

All scripts:
- Download latest release from GitHub (or specific version)
- Extract binaries
- Set up PATH
- Configure platform-specific daemon auto-start
- Include error handling and colored output
