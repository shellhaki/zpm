package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const addr = "127.0.0.1:4848"

var safeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func timeNow() int64 {
	return time.Now().Unix()
}

func Send(conn net.Conn, res Response) {
	json.NewEncoder(conn).Encode(res)
}

func CleanName(name string) string {
	name = strings.TrimSpace(name)
	name = safeName.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}

func LogPath(name string) string {
	return filepath.Join(LogsDir(), name+".log")
}

func IsRunning(pid int) bool {
	return processRunning(pid)
}

func Defaults(process Process) Process {
	if process.RestartDelay <= 0 {
		process.RestartDelay = 1000
	}
	if process.MaxRestarts == 0 {
		process.MaxRestarts = -1
	}
	if process.LogMaxBytes <= 0 {
		process.LogMaxBytes = 10 * 1024 * 1024
	}
	if process.LogBackups <= 0 {
		process.LogBackups = 5
	}
	if process.Env == nil {
		process.Env = map[string]string{}
	}
	return process
}

func RotateLog(path string, maxBytes int64, backups int) error {
	if maxBytes <= 0 || backups <= 0 {
		return nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() < maxBytes {
		return nil
	}
	os.Remove(fmt.Sprintf("%s.%d", path, backups))
	for i := backups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", path, i)
		newPath := fmt.Sprintf("%s.%d", path, i+1)
		os.Rename(oldPath, newPath)
	}
	return os.Rename(path, path+".1")
}

func EnvList(process Process) []string {
	env := os.Environ()
	envMap := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			envMap[key] = value
		}
	}
	if process.EnvName != "" {
		envMap["NODE_ENV"] = process.EnvName
		envMap["BUN_ENV"] = process.EnvName
		envMap["APP_ENV"] = process.EnvName
	}
	for key, value := range process.Env {
		envMap[key] = value
	}
	out := make([]string, 0, len(envMap))
	for key, value := range envMap {
		out = append(out, key+"="+value)
	}
	return out
}

func StartProcesses(req Request) ([]Process, error) {
	instances := req.Instances
	if instances <= 0 {
		instances = 1
	}
	processes := []Process{}
	for i := 0; i < instances; i++ {
		name := CleanName(req.Name)
		if instances > 1 {
			name = CleanName(fmt.Sprintf("%s-%d", req.Name, i))
		}
		process, err := StartProcess(Process{
			Name:          name,
			Command:       req.Command,
			Cwd:           req.Cwd,
			Env:           req.Env,
			EnvName:       req.EnvName,
			AutoRestart:   req.AutoRestart,
			RestartDelay:  req.RestartDelay,
			MaxRestarts:   req.MaxRestarts,
			HealthCommand: req.HealthCommand,
			LogMaxBytes:   req.LogMaxBytes,
			LogBackups:    req.LogBackups,
		})
		if err != nil {
			return processes, err
		}
		processes = append(processes, process)
	}
	return processes, nil
}

func StartProcess(input Process) (Process, error) {
	input.Name = CleanName(input.Name)
	input.Command = strings.TrimSpace(input.Command)
	input.Cwd = strings.TrimSpace(input.Cwd)

	if input.Command == "" {
		process, err := ResolveName(input.Name)
		if err != nil {
			return Process{}, err
		}
		if input.Cwd != "" {
			process.Cwd = input.Cwd
		}
		return LaunchProcess(process, false)
	}
	if input.Name == "" {
		return Process{}, errors.New("process name required")
	}
	if input.Cwd == "" {
		input.Cwd = "."
	}
	absCwd, err := filepath.Abs(input.Cwd)
	if err != nil {
		return Process{}, err
	}
	info, err := os.Stat(absCwd)
	if err != nil {
		return Process{}, err
	}
	if !info.IsDir() {
		return Process{}, errors.New("cwd is not a directory")
	}
	input.Cwd = absCwd

	existing, ok := GetProcess(input.Name)
	if ok && existing.Status == "running" && IsRunning(existing.Pid) {
		return Process{}, errors.New("process already running")
	}
	input = Defaults(input)
	input.Status = "stopped"
	input.LogPath = LogPath(input.Name)
	return LaunchProcess(input, false)
}

func LaunchProcess(process Process, restarted bool) (Process, error) {
	process = Defaults(process)
	if process.Status == "running" && IsRunning(process.Pid) {
		return Process{}, errors.New("process already running")
	}
	err := os.MkdirAll(LogsDir(), 0755)
	if err != nil {
		return Process{}, err
	}
	process.LogPath = LogPath(process.Name)
	err = RotateLog(process.LogPath, process.LogMaxBytes, process.LogBackups)
	if err != nil {
		return Process{}, err
	}
	logFile, err := os.OpenFile(process.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return Process{}, err
	}

	cmd := shellCommand(process.Command)
	cmd.Dir = process.Cwd
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.Env = EnvList(process)
	setDetached(cmd)

	err = cmd.Start()
	if err != nil {
		logFile.Close()
		return Process{}, err
	}

	process.Pid = cmd.Process.Pid
	process.Status = "running"
	process.StartedAt = timeNow()
	process.StoppedAt = 0
	process.ExitCode = 0
	process.Healthy = true
	process.LastError = ""
	if restarted {
		process.RestartCount++
	}
	err = UpsertProcess(process)
	if err != nil {
		logFile.Close()
		return Process{}, err
	}

	go MonitorProcess(process.Name, cmd, logFile)
	return process, nil
}

func MonitorProcess(name string, cmd *exec.Cmd, logFile *os.File) {
	err := cmd.Wait()
	logFile.Close()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	process, ok := GetProcess(name)
	if !ok {
		return
	}
	if process.Pid != cmd.Process.Pid {
		return
	}
	if process.Status == "stopping" || process.Status == "stopped" {
		UpdateProcess(name, func(p Process) Process {
			p.Pid = 0
			p.Status = "stopped"
			p.StoppedAt = timeNow()
			p.ExitCode = exitCode
			return p
		})
		return
	}
	if process.AutoRestart && (process.MaxRestarts < 0 || process.RestartCount < process.MaxRestarts) {
		time.Sleep(time.Duration(process.RestartDelay) * time.Millisecond)
		process.Pid = 0
		process.Status = "restarting"
		process.ExitCode = exitCode
		process.StoppedAt = timeNow()
		LaunchProcess(process, true)
		return
	}
	UpdateProcess(name, func(p Process) Process {
		p.Pid = 0
		p.Status = "crashed"
		if exitCode == 0 {
			p.Status = "stopped"
		}
		p.StoppedAt = timeNow()
		p.ExitCode = exitCode
		p.Healthy = false
		return p
	})
}

func StopProcess(name string) (Process, error) {
	process, err := ResolveName(CleanName(name))
	if err != nil {
		return Process{}, err
	}
	if process.Status != "running" || !IsRunning(process.Pid) {
		process.Status = "stopped"
		process.Pid = 0
		process.StoppedAt = timeNow()
		err = UpsertProcess(process)
		return process, err
	}
	UpdateProcess(process.Name, func(p Process) Process {
		p.Status = "stopping"
		return p
	})
	stopPid(process.Pid, false)
	for i := 0; i < 30; i++ {
		if !IsRunning(process.Pid) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if IsRunning(process.Pid) {
		stopPid(process.Pid, true)
	}
	process.Status = "stopped"
	process.Pid = 0
	process.StoppedAt = timeNow()
	err = UpsertProcess(process)
	return process, err
}

func StopTargets(name string) ([]Process, error) {
	targets, err := ResolveTargets(name)
	if err != nil {
		return nil, err
	}
	stopped := []Process{}
	for _, target := range targets {
		process, err := StopProcess(target.Name)
		if err != nil {
			return stopped, err
		}
		stopped = append(stopped, process)
	}
	return stopped, nil
}

func RestartProcess(name string) (Process, error) {
	process, err := StopProcess(name)
	if err != nil {
		return Process{}, err
	}
	return LaunchProcess(process, false)
}

func RestartTargets(name string) ([]Process, error) {
	targets, err := ResolveTargets(name)
	if err != nil {
		return nil, err
	}
	restarted := []Process{}
	for _, target := range targets {
		process, err := RestartProcess(target.Name)
		if err != nil {
			return restarted, err
		}
		restarted = append(restarted, process)
	}
	return restarted, nil
}

func PurgeProcess(name string) error {
	process, err := StopProcess(name)
	if err != nil {
		process, err = ResolveName(CleanName(name))
		if err != nil {
			return err
		}
	}
	err = DeleteProcess(process.Name)
	if err != nil {
		return err
	}
	if process.LogPath != "" {
		os.Remove(process.LogPath)
		for i := 1; i <= process.LogBackups; i++ {
			os.Remove(fmt.Sprintf("%s.%d", process.LogPath, i))
		}
	}
	return nil
}

func PurgeTargets(name string) ([]Process, error) {
	targets, err := ResolveTargets(name)
	if err != nil {
		return nil, err
	}
	purged := []Process{}
	for _, target := range targets {
		err := PurgeProcess(target.Name)
		if err != nil {
			return purged, err
		}
		purged = append(purged, target)
	}
	return purged, nil
}

func ReadRSS(pid int) int64 {
	file, err := os.Open(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, _ := strconv.ParseInt(fields[1], 10, 64)
				return value
			}
		}
	}
	return 0
}

func CheckHealth(process Process) (bool, string) {
	if !IsRunning(process.Pid) {
		return false, "process is not running"
	}
	if process.HealthCommand == "" {
		return true, ""
	}
	cmd := shellCommand(process.HealthCommand)
	cmd.Dir = process.Cwd
	cmd.Env = EnvList(process)
	err := cmd.Run()
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func RefreshStatuses() {
	for _, process := range ListProcesses() {
		if process.Status == "running" && !IsRunning(process.Pid) {
			SetProcessStopped(process.Name)
			continue
		}
		if process.Status == "running" {
			healthy, message := CheckHealth(process)
			UpdateProcess(process.Name, func(p Process) Process {
				p.Healthy = healthy
				p.LastError = message
				p.MemoryRssKB = ReadRSS(p.Pid)
				if p.StartedAt > 0 {
					p.UptimeSeconds = timeNow() - p.StartedAt
				} else {
					p.UptimeSeconds = 0
				}
				return p
			})
			if !healthy && process.AutoRestart {
				RestartProcess(process.Name)
			}
		}
	}
}

func HealthLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		RefreshStatuses()
	}
}

func HandleConnection(conn net.Conn) error {
	defer conn.Close()

	var req Request
	err := json.NewDecoder(conn).Decode(&req)
	if err != nil {
		Send(conn, Response{Success: false, Message: "bad request"})
		return err
	}

	switch req.Action {
	case "start":
		processes, err := StartProcesses(req)
		if err != nil {
			Send(conn, Response{Success: false, Message: err.Error(), Processes: processes})
			return err
		}
		res := Response{Success: true, Message: "started", Processes: processes}
		if len(processes) > 0 {
			res.Name = processes[0].Name
			res.Pid = processes[0].Pid
			res.Status = processes[0].Status
			res.LogPath = processes[0].LogPath
		}
		Send(conn, res)
	case "stop":
		processes, err := StopTargets(req.Name)
		if err != nil {
			Send(conn, Response{Success: false, Message: err.Error()})
			return err
		}
		res := Response{Success: true, Message: "stopped", Processes: processes}
		if len(processes) > 0 {
			res.Name = processes[0].Name
			res.Status = processes[0].Status
			res.LogPath = processes[0].LogPath
		}
		Send(conn, res)
	case "restart":
		processes, err := RestartTargets(req.Name)
		if err != nil {
			Send(conn, Response{Success: false, Message: err.Error()})
			return err
		}
		res := Response{Success: true, Message: "restarted", Processes: processes}
		if len(processes) > 0 {
			res.Name = processes[0].Name
			res.Pid = processes[0].Pid
			res.Status = processes[0].Status
			res.LogPath = processes[0].LogPath
		}
		Send(conn, res)
	case "purge":
		processes, err := PurgeTargets(req.Name)
		if err != nil {
			Send(conn, Response{Success: false, Message: err.Error()})
			return err
		}
		Send(conn, Response{Success: true, Message: "purged", Processes: processes})
	case "list", "status":
		RefreshStatuses()
		Send(conn, Response{Success: true, Processes: ListProcesses()})
	case "daemon-stop":
		Send(conn, Response{Success: true, Message: "daemon stopped"})
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
	default:
		Send(conn, Response{Success: false, Message: "unknown action"})
	}
	return nil
}

func main() {
	err := os.MkdirAll(DataDir(), 0755)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(PidPath(), []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		panic(err)
	}
	defer os.Remove(PidPath())

	err = LoadRegistry()
	if err != nil {
		panic(err)
	}
	RefreshStatuses()
	go HealthLoop()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	fmt.Println("daemon started and running on " + addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go HandleConnection(conn)
	}
}
