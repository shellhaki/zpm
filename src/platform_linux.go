//go:build linux

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

func startupUnitPath() string {
	return startupUnitPathFor("zpmd.service")
}

func startupUnitPathFor(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", name)
}

func installStartup() error {
	daemon, err := FindDaemon()
	if err != nil {
		return err
	}
	unitPath := startupUnitPath()
	err = os.MkdirAll(filepath.Dir(unitPath), 0755)
	if err != nil {
		return err
	}
	unit := "[Unit]\nDescription=ZPM daemon\nAfter=network.target\n\n[Service]\nType=simple\nExecStart=" + daemon + "\nRestart=always\nRestartSec=2\n\n[Install]\nWantedBy=default.target\n"
	err = os.WriteFile(unitPath, []byte(unit), 0644)
	if err != nil {
		return err
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return exec.Command("systemctl", "--user", "enable", "--now", "zpmd.service").Run()
}

func uninstallStartup() error {
	for _, service := range []string{"zpmd.service", "zpm.service"} {
		exec.Command("systemctl", "--user", "disable", "--now", service).Run()
	}
	err := removeExistingPaths(
		startupUnitPathFor("zpmd.service"),
		startupUnitPathFor("zpm.service"),
	)
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return err
}

func uninstallInstallArtifacts() error {
	paths := knownExecutablePaths("zpm", "zpmd")
	err := removeExistingPaths(paths...)
	if cleanupErr := removeShellPathEntries([]string{".bashrc", ".zshrc", ".bash_profile"}, []string{".local/bin"}); cleanupErr != nil {
		err = errors.Join(err, cleanupErr)
	}
	return err
}

func uninstallInstallArtifactNote() string {
	return "Restart your shell or reload your profile so PATH changes take effect."
}
