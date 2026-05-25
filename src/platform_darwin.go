//go:build darwin

package main

import (
	"errors"
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
	return startupPlistPathFor("com.zpm.daemon.plist")
}

func startupPlistPathFor(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", name)
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
<key>Label</key><string>com.zpm.daemon</string>
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
	for _, item := range []struct {
		label string
		path  string
	}{
		{"com.zpm.daemon", startupPlistPathFor("com.zpm.daemon.plist")},
		{"dev.zpm.daemon", startupPlistPathFor("dev.zpm.daemon.plist")},
	} {
		exec.Command("launchctl", "unload", item.path).Run()
		exec.Command("launchctl", "remove", item.label).Run()
	}
	return removeExistingPaths(
		startupPlistPathFor("com.zpm.daemon.plist"),
		startupPlistPathFor("dev.zpm.daemon.plist"),
	)
}

func uninstallInstallArtifacts() error {
	paths := knownExecutablePaths("zpm", "zpmd")
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".zpm"))
	}
	err := removeExistingPaths(paths...)
	if cleanupErr := removeShellPathEntries([]string{".zshrc", ".bash_profile", ".bashrc"}, []string{".local/bin"}); cleanupErr != nil {
		err = errors.Join(err, cleanupErr)
	}
	return err
}

func uninstallInstallArtifactNote() string {
	return "Restart your shell or reload your profile so PATH changes take effect."
}
