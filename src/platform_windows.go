//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"syscall"
)

func setDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func stopPid(pid int, force bool) {
	if pid <= 0 {
		return
	}
	exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
}

func stopDaemonByName(force bool) {
	exec.Command("taskkill", "/IM", "zpmd.exe", "/T", "/F").Run()
}

func installStartup() error {
	daemon, err := FindDaemon()
	if err != nil {
		return err
	}
	return exec.Command("schtasks", "/Create", "/TN", "zpm", "/TR", daemon, "/SC", "ONLOGON", "/F").Run()
}

func uninstallStartup() error {
	return exec.Command("schtasks", "/Delete", "/TN", "zpm", "/F").Run()
}
