//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	exec.Command("schtasks", "/Delete", "/TN", "zpm", "/F").Run()
	return exec.Command("schtasks", "/Create", "/TN", "ZPM Daemon", "/TR", daemon, "/SC", "ONLOGON", "/F").Run()
}

func uninstallStartup() error {
	for _, task := range []string{"ZPM Daemon", "zpm"} {
		exec.Command("schtasks", "/End", "/TN", task).Run()
		exec.Command("schtasks", "/Delete", "/TN", task, "/F").Run()
	}
	return nil
}

func uninstallInstallArtifacts() error {
	var errs []error
	exe, _ := os.Executable()
	exeDir := ""
	if exe != "" {
		exeDir = filepath.Dir(exe)
	}

	localRoot := ""
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		localRoot = filepath.Join(localAppData, "zpm")
	}

	removeDirsFromUserPath([]string{
		filepath.Join(localRoot, "bin"),
		exeDir,
	})

	paths := []string{}
	if exeDir != "" {
		paths = append(paths,
			filepath.Join(exeDir, "zpm.exe"),
			filepath.Join(exeDir, "zpmd.exe"),
			filepath.Join(exeDir, "zpp.exe"),
		)
	}
	if localRoot != "" && !pathContains(localRoot, exe) {
		paths = append(paths, localRoot)
	}
	if err := removeExistingPaths(paths...); err != nil {
		errs = append(errs, err)
	}

	delayed := []string{}
	if exe != "" {
		delayed = append(delayed, exe)
		if exeDir != "" {
			delayed = append(delayed, filepath.Join(exeDir, "zpmd.exe"))
		}
	}
	if localRoot != "" && pathContains(localRoot, exe) {
		delayed = append(delayed, localRoot)
	}
	if len(delayed) > 0 {
		if err := scheduleWindowsRemoval(delayed); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func uninstallInstallArtifactNote() string {
	return "Windows deletes the running zpm.exe after this command exits. Restart PowerShell so PATH changes take effect."
}

func scheduleWindowsRemoval(paths []string) error {
	commands := []string{"ping 127.0.0.1 -n 3 >NUL"}
	for _, path := range uniqueCleanPaths(paths) {
		if path == "" {
			continue
		}
		commands = append(commands,
			"del /F /Q "+cmdQuote(path)+" 2>NUL",
			"rmdir /S /Q "+cmdQuote(path)+" 2>NUL",
		)
	}
	cmd := exec.Command("cmd", "/C", strings.Join(commands, " & "))
	setDetached(cmd)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Process.Release()
}

func removeDirsFromUserPath(dirs []string) {
	cleaned := []string{}
	for _, dir := range uniqueCleanPaths(dirs) {
		if dir != "" {
			cleaned = append(cleaned, dir)
		}
	}
	if len(cleaned) == 0 {
		return
	}

	psDirs := []string{}
	for _, dir := range cleaned {
		psDirs = append(psDirs, psQuote(dir))
	}
	script := "$targets=@(" + strings.Join(psDirs, ",") + ");" +
		"$path=[Environment]::GetEnvironmentVariable('Path','User');" +
		"if($path){" +
		"$parts=$path -split ';' | Where-Object {" +
		"$part=$_.Trim();" +
		"if(-not $part){return $false};" +
		"$keep=$true;" +
		"foreach($target in $targets){if($part.TrimEnd('\\') -ieq $target.TrimEnd('\\')){$keep=$false}}" +
		"$keep" +
		"};" +
		"[Environment]::SetEnvironmentVariable('Path',($parts -join ';'),'User')" +
		"}"
	exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script).Run()
}

func pathContains(parent string, child string) bool {
	if parent == "" || child == "" {
		return false
	}
	parent = strings.ToLower(filepath.Clean(parent))
	child = strings.ToLower(filepath.Clean(child))
	return child == parent || strings.HasPrefix(child, parent+string(os.PathSeparator))
}

func uniqueCleanPaths(paths []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		key := strings.ToLower(path)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, path)
	}
	return out
}

func cmdQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func psQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
