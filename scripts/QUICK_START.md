# ZPM Quick Install Reference

## Install in One Command

### Linux/macOS (Bash)
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash
```

### Windows (PowerShell as Administrator)
```powershell
iex (curl.exe -UseBasicParsing https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install-windows.ps1)
```

---

## What Gets Installed

| Component | Description |
|-----------|-------------|
| **zpm** | Command-line interface for managing processes |
| **zpmd** | Background daemon (auto-starts on boot) |
| **PATH** | Binary location added to your system PATH |

---

## After Installation

### Start the Daemon
```bash
zpmd  # or let it auto-start on boot
```

### Try ZPM
```bash
zpm --help              # See available commands
zpm list                # List managed processes
zpm start script.js     # Start managing a process
```

---

## Daemon Auto-Start Status

| Platform | Status | How to Enable |
|----------|--------|---------------|
| **Linux** | systemd user service | `systemctl --user enable zpmd` |
| **macOS** | LaunchAgent | `launchctl load ~/Library/LaunchAgents/com.zpm.daemon.plist` |
| **Windows** | Task Scheduler | Set up automatically if installer run as Admin |

---

## Specify Version

```bash
# Linux/macOS with specific version
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash -s v0.1.0

# Windows with specific version (PowerShell)
# Edit the script or download manually from GitHub Releases
```

---

## Supported Architectures

✅ **Linux**: amd64, arm64  
✅ **macOS**: amd64 (Intel), arm64 (Apple Silicon)  
✅ **Windows**: amd64  

Architecture is auto-detected and the correct binary is downloaded.

---

## Troubleshooting

**zpm/zpmd command not found?**
- Linux/macOS: `source ~/.bashrc` or `source ~/.zshrc`
- Windows: Restart your terminal

**Daemon won't start?**
- Linux: `systemctl --user status zpmd`
- macOS: `launchctl list | grep zpm`
- Windows: Check Task Scheduler for "ZPM Daemon"

**More help?** See `scripts/README.md` for detailed installation guide and troubleshooting.

---

## Uninstall

```bash
# Linux
systemctl --user disable zpmd
rm ~/.local/bin/{zpm,zpmd}

# macOS
launchctl unload ~/Library/LaunchAgents/com.zpm.daemon.plist
rm -rf ~/.local/bin/{zpm,zpmd} ~/.zpm ~/Library/LaunchAgents/com.zpm.daemon.plist

# Windows (PowerShell as Admin)
Unregister-ScheduledTask -TaskName "ZPM Daemon" -Confirm:$false
Remove-Item "$env:LOCALAPPDATA\zpm" -Recurse
# Then manually remove from PATH environment variables
```
