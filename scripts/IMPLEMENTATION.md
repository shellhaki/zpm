# ZPM Installation Scripts - Implementation Summary

## Overview

Created a comprehensive, cross-platform installation system for ZPM (Zen Process Manager) that automatically handles:
- OS and architecture detection
- Downloading correct release binaries from GitHub
- Installing both `zpm` (CLI) and `zpmd` (daemon)
- PATH configuration
- Platform-specific daemon auto-start setup

## Files Created

### 1. **install.sh** - Universal Entry Point
- Detects the operating system (Linux, macOS, Windows)
- Routes to the appropriate platform-specific installer
- Supports passing version parameter
- Works with simple curl one-liner

**Usage:**
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash
```

### 2. **install-linux.sh** - Linux Installer
- Auto-detects architecture (amd64 or arm64)
- Downloads from GitHub releases: `zpm-linux-{arch}.tar.gz`
- Installs binaries to `~/.local/bin/`
- Adds PATH export to `~/.bashrc` or `~/.zshrc`
- Creates systemd user service (`~/.config/systemd/user/zpmd.service`)
- Provides instructions for enabling zpmd on startup

**Features:**
- Handles both amd64 and arm64 architectures
- Detects shell type (bash/zsh) and updates appropriate rc file
- Creates systemd service automatically
- Color-coded output for clarity
- Comprehensive error handling

### 3. **install-macos.sh** - macOS Installer
- Auto-detects architecture (amd64 or arm64)
- Downloads from GitHub releases: `zpm-darwin-{arch}.tar.gz`
- Installs binaries to `~/.local/bin/`
- Adds PATH export to `~/.zshrc` or `~/.bash_profile`
- Creates LaunchAgent (`~/Library/LaunchAgents/com.zpm.daemon.plist`)
- Creates log directory (`~/.zpm/`)

**Features:**
- Native macOS LaunchAgent for daemon management
- Automatic log file routing
- Shell detection (zsh is modern default, fallback to bash)
- Handles both Intel (amd64) and Apple Silicon (arm64)
- Log directory automatically created

### 4. **install-windows.ps1** - Windows PowerShell Installer
- Detects architecture (amd64 only for now)
- Downloads from GitHub releases: `zpm-windows-amd64.tar.gz`
- Installs binaries to `%LOCALAPPDATA%\zpm\bin`
- Adds to User PATH environment variable
- Creates Task Scheduler task (if run as Administrator)
- Proper error handling for tar extraction

**Features:**
- PowerShell native implementation
- Automatic PATH environment variable update
- Task Scheduler integration for automatic startup on logon
- Works on Windows 10+ (has native tar support)
- Graceful handling when not run as Administrator

### 5. **README.md** - Installation Guide
- Quick start commands for all platforms
- Detailed installation explanations per platform
- Troubleshooting guide
- Uninstall instructions
- Manual build from source instructions
- Custom installation directory support

## Architecture & Design

### Single Responsibility
Each script has a clear purpose:
- `install.sh`: OS detection and routing
- Platform-specific scripts: Handle OS-specific paths, services, environment setup

### Error Handling
All scripts include:
- Exit on error (`set -e` for bash)
- URL validation and error messages
- Binary verification before installation
- Helpful error messages with context

### Daemon Management
Platforms use native service management:
- **Linux**: systemd (modern, user-level services)
- **macOS**: LaunchAgent (native macOS way)
- **Windows**: Task Scheduler (built-in Windows scheduler)

### Architecture Auto-Detection
- **Linux**: Detects amd64/arm64 via `uname -m`
- **macOS**: Detects amd64/arm64 via `uname -m`
- **Windows**: Detects via `PROCESSOR_ARCHITECTURE` environment variable

## Key Implementation Details

### GitHub Release Download
All scripts use GitHub API to fetch the latest (or specific) release:
```
https://api.github.com/repos/shellhaki/zpm/releases/latest
or
https://api.github.com/repos/shellhaki/zpm/releases/tags/{version}
```

Assets are matched by filename pattern:
- Linux: `zpm-linux-{arch}.tar.gz`
- macOS: `zpm-darwin-{arch}.tar.gz`
- Windows: `zpm-windows-amd64.tar.gz`

### PATH Management
Different strategies per platform:

**Linux/macOS:**
- Export statement added to shell RC file
- Only added if not already present
- User is prompted to source the file

**Windows:**
- User PATH environment variable updated directly
- Persists across terminal restarts
- Requires no manual sourcing

### Installation Directories
- **Linux/macOS**: `~/.local/bin/` (user-local, no sudo needed)
- **Windows**: `%LOCALAPPDATA%\zpm\bin` (user-specific, no admin needed)

### Daemon Startup

**Linux (systemd):**
```bash
systemctl --user enable zpmd
systemctl --user start zpmd
```

**macOS (LaunchAgent):**
```bash
launchctl load ~/Library/LaunchAgents/com.zpm.daemon.plist
```

**Windows (Task Scheduler):**
- Runs on user logon automatically (if admin ran installer)
- Or manual: `zpmd` command

## Usage Examples

### Simplest: One-liner Installation

**Linux/macOS:**
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash
```

**Windows (PowerShell as Administrator):**
```powershell
iex (curl.exe -UseBasicParsing https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install-windows.ps1)
```

### With Specific Version

**Linux/macOS:**
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash -s v0.1.0
```

### With Custom Install Directory

**Linux/macOS:**
```bash
curl -sL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | bash -s latest /usr/local/bin
```

## Security Considerations

1. **HTTPS only**: All downloads are from GitHub HTTPS endpoints
2. **Verification**: Scripts verify binaries exist after extraction
3. **No sudo required**: User-level installations (except Windows Task Scheduler)
4. **Clean extraction**: Uses temporary directories with proper cleanup
5. **No hardcoded paths**: Uses environment variables and user home directory

## Dependencies

**Minimal external dependencies:**
- `curl`: For downloading (widely available)
- `tar`: For extraction (built-in on all platforms)
- Basic shell utilities: `grep`, `sed`, `mkdir`, `cp`, `chmod`
- **Windows**: PowerShell (built-in), tar in Windows 10+

## Testing Recommendations

1. **Test on each platform:**
   - Ubuntu/Debian (Linux amd64)
   - Raspberry Pi OS (Linux arm64)
   - macOS Intel (amd64)
   - macOS Apple Silicon (arm64)
   - Windows 10/11 (amd64)

2. **Test scenarios:**
   - First-time installation
   - Re-installation (latest version)
   - Specific version installation
   - Custom install directory
   - PATH updates in different shells

3. **Verify daemon startup:**
   - Linux: `systemctl --user status zpmd`
   - macOS: `launchctl list | grep zpm`
   - Windows: `Get-ScheduledTask -TaskName "ZPM Daemon"`

## Future Enhancements

Possible improvements:
1. Add Windows arm64 support (if build pipeline supports it)
2. Add deb/rpm package generation (Linux)
3. Add Homebrew tap for macOS
4. Add Windows Store/scoop support
5. Add checksum verification for security
6. Add auto-update capability

## File Structure

```
scripts/
├── install.sh              # Main entry point
├── install-linux.sh        # Linux-specific
├── install-macos.sh        # macOS-specific
├── install-windows.ps1     # Windows PowerShell
└── README.md               # Complete guide
```

## Integration with Release Workflow

These scripts work with the existing release.yml workflow:
- They pull from GitHub releases created by the CI/CD pipeline
- Support multiple architectures automatically
- No changes needed to the build process
- Can be invoked immediately after release creation
