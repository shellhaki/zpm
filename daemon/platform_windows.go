//go:build windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func shellCommand(command string) *exec.Cmd {
	return exec.Command("cmd", "/C", command)
}

func setDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func processRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func stopPid(pid int, force bool) {
	if pid <= 0 {
		return
	}
	exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
}
