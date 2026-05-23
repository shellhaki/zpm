# ZPM (Zen Process Manager) Installer for Windows
# Downloads and installs both zpm and zpmd
# Run as Administrator for full functionality

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\zpm\bin"
)

# Color output functions
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Error-Color { Write-Host $args -ForegroundColor Red }
function Write-Warning-Color { Write-Host $args -ForegroundColor Yellow }

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-Error-Color "Error: Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get download URL from GitHub API
function Get-DownloadUrl {
    param(
        [string]$Version,
        [string]$Architecture
    )
    
    $repo = "shellhaki/zpm"
    $github_api = "https://api.github.com/repos/$repo/releases"
    
    try {
        $release_url = if ($Version -eq "latest") {
            "$github_api/latest"
        } else {
            "$github_api/tags/$Version"
        }
        
        $release_info = Invoke-WebRequest -Uri $release_url -UseBasicParsing -ErrorAction Stop
        $release_json = $release_info.Content | ConvertFrom-Json
        
        $download_url = $release_json.assets | 
            Where-Object { $_.name -match "zpm-windows-$Architecture\.tar\.gz" } |
            Select-Object -ExpandProperty browser_download_url -First 1
        
        if (-not $download_url) {
            Write-Error-Color "Error: Could not find release for windows-$Architecture"
            exit 1
        }
        
        return $download_url
    }
    catch {
        Write-Error-Color "Error fetching release: $_"
        exit 1
    }
}

# Main installation
function Install-ZPM {
    Write-Warning-Color "=== ZPM Installer for Windows ==="
    
    $arch = Get-Architecture
    Write-Success "✓ Detected architecture: windows-$arch"
    
    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    Write-Success "✓ Install directory: $InstallDir"
    
    # Get download URL
    Write-Warning-Color "Fetching release information..."
    $download_url = Get-DownloadUrl -Version $Version -Architecture $arch
    Write-Success "✓ Download URL: $download_url"
    
    # Create temporary directory
    $temp_dir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    
    try {
        # Download
        Write-Warning-Color "Downloading ZPM $Version..."
        $zip_path = Join-Path $temp_dir "zpm.tar.gz"
        
        try {
            Invoke-WebRequest -Uri $download_url -OutFile $zip_path -UseBasicParsing -ErrorAction Stop
            Write-Success "✓ Downloaded successfully"
        }
        catch {
            Write-Error-Color "✗ Failed to download ZPM: $_"
            exit 1
        }
        
        # Extract tar.gz (using tar command available in Windows 10+)
        Write-Warning-Color "Extracting binaries..."
        
        try {
            # Windows 10+ has tar built-in
            & tar -xzf $zip_path -C $temp_dir
        }
        catch {
            # Fallback for older Windows
            Write-Error-Color "Error: tar command not found. Please install 7-Zip or update Windows."
            exit 1
        }
        
        $extracted_dir = Get-ChildItem -Path $temp_dir -Directory -Filter "zpm-windows-*" | Select-Object -First 1
        
        if (-not $extracted_dir) {
            Write-Error-Color "✗ Failed to extract archive"
            exit 1
        }
        
        # Verify binaries exist
        $zpm_exe = Join-Path $extracted_dir.FullName "zpm.exe"
        $zpmd_exe = Join-Path $extracted_dir.FullName "zpmd.exe"
        
        if (-not (Test-Path $zpm_exe) -or -not (Test-Path $zpmd_exe)) {
            Write-Error-Color "✗ Required binaries not found in archive"
            exit 1
        }
        
        # Copy binaries
        Copy-Item $zpm_exe $InstallDir -Force
        Copy-Item $zpmd_exe $InstallDir -Force
        Write-Success "✓ Binaries installed"
        
        # Add to PATH
        $current_path = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($current_path -notlike "*$InstallDir*") {
            [Environment]::SetEnvironmentVariable("Path", "$current_path;$InstallDir", "User")
            Write-Success "✓ Added to PATH (User environment variable)"
            Write-Warning-Color "Note: Restart your terminal for PATH changes to take effect"
        }
        
        # Create Task Scheduler task for zpmd (requires admin)
        $is_admin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        
        if ($is_admin) {
            Write-Warning-Color "Setting up Task Scheduler for zpmd..."
            
            # Create task to run zpmd on startup
            $action = New-ScheduledTaskAction -Execute (Join-Path $InstallDir "zpmd.exe")
            $trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
            $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
            
            $task_name = "ZPM Daemon"
            
            # Remove existing task if it exists
            Get-ScheduledTask -TaskName $task_name -ErrorAction SilentlyContinue | Unregister-ScheduledTask -Confirm:$false
            
            # Create new task
            Register-ScheduledTask `
                -TaskName $task_name `
                -Action $action `
                -Trigger $trigger `
                -Settings $settings `
                -Description "ZPM (Zen Process Manager) daemon - starts on logon" `
                -ErrorAction SilentlyContinue | Out-Null
            
            Write-Success "✓ Created Task Scheduler entry: $task_name"
            Write-Warning-Color "zpmd will start automatically on next logon"
        }
        else {
            Write-Warning-Color "⚠ Run as Administrator to set up automatic zpmd startup"
            Write-Warning-Color "To start zpmd manually, run: zpmd"
        }
        
        Write-Host ""
        Write-Success "=== Installation Complete ==="
        Write-Warning-Color "Next steps:"
        Write-Host "1. Restart your terminal to load PATH changes"
        Write-Host "2. Start zpmd: zpmd (or wait for automatic startup on logon if admin ran this)"
        Write-Host "3. Test installation: zpm --help"
    }
    finally {
        # Cleanup temporary directory
        Remove-Item $temp_dir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Run installation
Install-ZPM
