//go:build darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func setDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func stopPid(pid int, force bool) {
	if pid <= 0 {
		return
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	syscall.Kill(pid, signal)
}

func stopDaemonByName(force bool) {
	if force {
		exec.Command("pkill", "-KILL", "-x", "zpmd").Run()
		return
	}
	exec.Command("pkill", "-TERM", "-x", "zpmd").Run()
}

func startupPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "dev.zpm.daemon.plist")
}

func installStartup() error {
	daemon, err := FindDaemon()
	if err != nil {
		return err
	}
	plistPath := startupPlistPath()
	err = os.MkdirAll(filepath.Dir(plistPath), 0755)
	if err != nil {
		return err
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>dev.zpm.daemon</string>
<key>ProgramArguments</key><array><string>` + daemon + `</string></array>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><true/>
</dict></plist>`
	err = os.WriteFile(plistPath, []byte(plist), 0644)
	if err != nil {
		return err
	}
	exec.Command("launchctl", "unload", plistPath).Run()
	return exec.Command("launchctl", "load", plistPath).Run()
}

func uninstallStartup() error {
	plistPath := startupPlistPath()
	exec.Command("launchctl", "unload", plistPath).Run()
	return os.Remove(plistPath)
}
