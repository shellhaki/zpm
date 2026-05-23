package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const addr = "127.0.0.1:4848"

type Request struct {
	Action        string            `json:"action"`
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Cwd           string            `json:"cwd"`
	Env           map[string]string `json:"env"`
	EnvName       string            `json:"env_name"`
	Instances     int               `json:"instances"`
	AutoRestart   bool              `json:"auto_restart"`
	RestartDelay  int               `json:"restart_delay"`
	MaxRestarts   int               `json:"max_restarts"`
	HealthCommand string            `json:"health_command"`
	LogMaxBytes   int64             `json:"log_max_bytes"`
	LogBackups    int               `json:"log_backups"`
}

type Process struct {
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Cwd           string            `json:"cwd"`
	Env           map[string]string `json:"env"`
	EnvName       string            `json:"env_name"`
	Pid           int               `json:"pid"`
	Status        string            `json:"status"`
	LogPath       string            `json:"log_path"`
	AutoRestart   bool              `json:"auto_restart"`
	RestartDelay  int               `json:"restart_delay"`
	MaxRestarts   int               `json:"max_restarts"`
	RestartCount  int               `json:"restart_count"`
	StartedAt     int64             `json:"started_at"`
	StoppedAt     int64             `json:"stopped_at"`
	ExitCode      int               `json:"exit_code"`
	HealthCommand string            `json:"health_command"`
	Healthy       bool              `json:"healthy"`
	LastError     string            `json:"last_error"`
	LogMaxBytes   int64             `json:"log_max_bytes"`
	LogBackups    int               `json:"log_backups"`
	MemoryRssKB   int64             `json:"memory_rss_kb"`
	UptimeSeconds int64             `json:"uptime_seconds"`
}

type Response struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Name      string    `json:"name"`
	Pid       int       `json:"pid"`
	Status    string    `json:"status"`
	LogPath   string    `json:"log_path"`
	Processes []Process `json:"processes"`
}

type Ecosystem struct {
	Apps []EcosystemApp `json:"apps"`
}

type EcosystemApp struct {
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Cwd           string            `json:"cwd"`
	Instances     int               `json:"instances"`
	Env           map[string]string `json:"env"`
	EnvProduction map[string]string `json:"env_production"`
	AutoRestart   *bool             `json:"auto_restart"`
	RestartDelay  int               `json:"restart_delay"`
	MaxRestarts   int               `json:"max_restarts"`
	HealthCommand string            `json:"health_command"`
	LogMaxBytes   int64             `json:"log_max_bytes"`
	LogBackups    int               `json:"log_backups"`
}

func DataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ".zpm"
	}
	return filepath.Join(dir, "zpm")
}

func DaemonLogPath() string {
	return filepath.Join(DataDir(), "daemon.log")
}

func DaemonPidPath() string {
	return filepath.Join(DataDir(), "daemon.pid")
}

func Send(req Request) (Response, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return Response{}, errors.New("daemon is not running")
	}
	defer conn.Close()

	err = json.NewEncoder(conn).Encode(req)
	if err != nil {
		return Response{}, err
	}

	var res Response
	err = json.NewDecoder(conn).Decode(&res)
	if err != nil {
		return Response{}, err
	}
	if !res.Success {
		return res, errors.New(res.Message)
	}
	return res, nil
}

func DaemonRunning() bool {
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func FindDaemon() (string, error) {
	if path := os.Getenv("ZPMD_PATH"); path != "" {
		return path, nil
	}
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "daemon", "zpmd"),
		filepath.Join(cwd, "..", "daemon", "zpmd"),
		"zpmd",
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidates = append([]string{
			filepath.Join(dir, "zpmd"),
			filepath.Join(dir, "..", "daemon", "zpmd"),
		}, candidates...)
	}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("zpmd binary not found")
}

func StartDaemon() error {
	if DaemonRunning() {
		fmt.Println("daemon already running")
		return nil
	}
	daemon, err := FindDaemon()
	if err != nil {
		return err
	}
	err = os.MkdirAll(DataDir(), 0755)
	if err != nil {
		return err
	}
	logFile, err := os.OpenFile(DaemonLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	cmd := exec.Command(daemon)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	setDetached(cmd)
	err = cmd.Start()
	if err != nil {
		logFile.Close()
		return err
	}
	err = cmd.Process.Release()
	if err != nil {
		logFile.Close()
		return err
	}
	logFile.Close()
	for i := 0; i < 20; i++ {
		if DaemonRunning() {
			fmt.Println("daemon started")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("daemon did not start")
}

func StopDaemon() error {
	if DaemonRunning() {
		res, err := Send(Request{Action: "daemon-stop"})
		if err == nil {
			fmt.Println(res.Message)
		}
		for i := 0; i < 20; i++ {
			if !DaemonRunning() {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	data, err := os.ReadFile(DaemonPidPath())
	if err == nil {
		pid, scanErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if scanErr == nil && pid > 0 {
			stopPid(pid, false)
		}
		os.Remove(DaemonPidPath())
	}
	stopDaemonByName(false)
	for i := 0; i < 20; i++ {
		if !DaemonRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	stopDaemonByName(true)
	for i := 0; i < 20; i++ {
		if !DaemonRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if DaemonRunning() {
		return errors.New("daemon did not stop")
	}
	fmt.Println("daemon not running")
	return nil
}

func ReloadDaemon() error {
	err := StopDaemon()
	if err != nil {
		return err
	}
	return StartDaemon()
}

func ParseBytes(value string) (int64, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	multiplier := int64(1)
	switch {
	case strings.HasSuffix(value, "kb"):
		multiplier = 1024
		value = strings.TrimSuffix(value, "kb")
	case strings.HasSuffix(value, "mb"):
		multiplier = 1024 * 1024
		value = strings.TrimSuffix(value, "mb")
	case strings.HasSuffix(value, "gb"):
		multiplier = 1024 * 1024 * 1024
		value = strings.TrimSuffix(value, "gb")
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return number * multiplier, nil
}

func ParseStart(args []string) (Request, bool, error) {
	req := Request{Action: "start", AutoRestart: true, MaxRestarts: -1, Env: map[string]string{}}
	var follow bool
	parts := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--follow", "-f":
			follow = true
		case "--name", "-n":
			if i+1 >= len(args) {
				return req, false, errors.New("missing process name")
			}
			req.Name = args[i+1]
			i++
		case "--env":
			if i+1 >= len(args) {
				return req, false, errors.New("missing env")
			}
			value := args[i+1]
			if strings.Contains(value, "=") {
				key, val, _ := strings.Cut(value, "=")
				req.Env[key] = val
			} else {
				req.EnvName = value
			}
			i++
		case "--instances", "-i":
			if i+1 >= len(args) {
				return req, false, errors.New("missing instances")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return req, false, err
			}
			req.Instances = value
			i++
		case "--no-autorestart":
			req.AutoRestart = false
		case "--restart-delay":
			if i+1 >= len(args) {
				return req, false, errors.New("missing restart delay")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return req, false, err
			}
			req.RestartDelay = value
			i++
		case "--max-restarts":
			if i+1 >= len(args) {
				return req, false, errors.New("missing max restarts")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return req, false, err
			}
			req.MaxRestarts = value
			i++
		case "--health":
			if i+1 >= len(args) {
				return req, false, errors.New("missing health command")
			}
			req.HealthCommand = args[i+1]
			i++
		case "--log-max-size":
			if i+1 >= len(args) {
				return req, false, errors.New("missing log max size")
			}
			value, err := ParseBytes(args[i+1])
			if err != nil {
				return req, false, err
			}
			req.LogMaxBytes = value
			i++
		case "--log-backups":
			if i+1 >= len(args) {
				return req, false, errors.New("missing log backups")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return req, false, err
			}
			req.LogBackups = value
			i++
		default:
			parts = append(parts, args[i])
		}
	}
	if len(parts) == 0 {
		return req, false, errors.New("command or process name required")
	}
	if req.Name == "" && len(parts) == 1 {
		req.Name = parts[0]
		return req, follow, nil
	}
	req.Command = strings.Join(parts, " ")
	return req, follow, nil
}

func Follow(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	buffer := make([]byte, 4096)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			fmt.Print(string(buffer[:n]))
		}
		if err == io.EOF {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if err != nil {
			return err
		}
	}
}

func PrintProcess(res Response) {
	if len(res.Processes) > 1 {
		for _, process := range res.Processes {
			fmt.Printf("%s %s pid=%d\n", res.Message, process.Name, process.Pid)
		}
		return
	}
	if res.Pid > 0 {
		fmt.Printf("%s %s pid=%d\n", res.Message, res.Name, res.Pid)
		return
	}
	if res.Name != "" {
		fmt.Printf("%s %s\n", res.Message, res.Name)
		return
	}
	fmt.Println(res.Message)
}

func PrintStatus(processes []Process) {
	for _, p := range processes {
		health := "ok"
		if !p.Healthy && p.Status == "running" {
			health = "bad"
		}
		fmt.Printf("%s\t%s\tpid=%d\tmem=%dkb\tuptime=%ds\trestarts=%d\thealth=%s\t%s\n", p.Name, p.Status, p.Pid, p.MemoryRssKB, p.UptimeSeconds, p.RestartCount, health, p.Command)
	}
}

func LoadEcosystem(path string, envName string) ([]Request, error) {
	if path == "" {
		path = "zpm.config.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ecosystem Ecosystem
	err = json.Unmarshal(data, &ecosystem)
	if err != nil {
		return nil, err
	}
	base, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	requests := []Request{}
	for _, app := range ecosystem.Apps {
		autoRestart := true
		if app.AutoRestart != nil {
			autoRestart = *app.AutoRestart
		}
		env := map[string]string{}
		for key, value := range app.Env {
			env[key] = value
		}
		if envName == "production" {
			for key, value := range app.EnvProduction {
				env[key] = value
			}
		}
		cwd := app.Cwd
		if cwd == "" {
			cwd = base
		}
		requests = append(requests, Request{
			Action:        "start",
			Name:          app.Name,
			Command:       app.Command,
			Cwd:           cwd,
			Env:           env,
			EnvName:       envName,
			Instances:     app.Instances,
			AutoRestart:   autoRestart,
			RestartDelay:  app.RestartDelay,
			MaxRestarts:   app.MaxRestarts,
			HealthCommand: app.HealthCommand,
			LogMaxBytes:   app.LogMaxBytes,
			LogBackups:    app.LogBackups,
		})
	}
	return requests, nil
}

func StartupInstall() error {
	StopDaemon()
	err := installStartup()
	if err != nil {
		return err
	}
	fmt.Println("startup installed")
	return nil
}

func StartupUninstall() error {
	err := uninstallStartup()
	if err != nil {
		return err
	}
	fmt.Println("startup removed")
	return nil
}

func Usage() {
	fmt.Println("zpp daemon start|stop|reload")
	fmt.Println("zpp startup install|uninstall")
	fmt.Println("zpp start \"bun index\" --name app --env production --instances 2 --follow")
	fmt.Println("zpp start app")
	fmt.Println("zpp stop app")
	fmt.Println("zpp restart app")
	fmt.Println("zpp purge app")
	fmt.Println("zpp status")
	fmt.Println("zpp ecosystem start [zpm.config.json] --env production")
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		Usage()
		os.Exit(1)
	}

	switch args[0] {
	case "daemon":
		if len(args) < 2 {
			Usage()
			os.Exit(1)
		}
		var err error
		switch args[1] {
		case "start":
			err = StartDaemon()
		case "stop":
			err = StopDaemon()
		case "reload", "restart":
			err = ReloadDaemon()
		default:
			Usage()
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "startup":
		if len(args) < 2 {
			Usage()
			os.Exit(1)
		}
		var err error
		switch args[1] {
		case "install":
			err = StartupInstall()
		case "uninstall", "remove":
			err = StartupUninstall()
		default:
			Usage()
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "ecosystem":
		if len(args) < 2 || args[1] != "start" {
			Usage()
			os.Exit(1)
		}
		path := ""
		envName := ""
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--env":
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "missing env")
					os.Exit(1)
				}
				envName = args[i+1]
				i++
			default:
				path = args[i]
			}
		}
		requests, err := LoadEcosystem(path, envName)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, req := range requests {
			res, err := Send(req)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			PrintProcess(res)
		}
	case "start":
		if len(args) >= 2 && args[1] == "daemon" {
			err := StartDaemon()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
		req, follow, err := ParseStart(args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		req.Cwd = cwd
		res, err := Send(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		PrintProcess(res)
		if follow {
			if len(res.Processes) > 0 {
				err = Follow(res.Processes[0].LogPath)
			} else {
				err = Follow(res.LogPath)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	case "stop", "restart", "purge":
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		res, err := Send(Request{Action: args[0], Name: name})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		PrintProcess(res)
	case "list", "status":
		res, err := Send(Request{Action: "status"})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		PrintStatus(res.Processes)
	default:
		Usage()
		os.Exit(1)
	}
}
