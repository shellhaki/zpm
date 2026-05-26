package main

import (
	"bufio"
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

	"github.com/shellhaki/zpmcli/publish"
)

const addr = "127.0.0.1:4848"

const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorGreen  = "\033[38;5;76m"
	ColorRed    = "\033[38;5;196m"
	ColorYellow = "\033[38;5;214m"
	ColorCyan   = "\033[38;5;44m"
	ColorGray   = "\033[38;5;244m"
)

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
		fmt.Printf("  %s•%s %sDaemon is already running.%s\n", ColorCyan, ColorReset, ColorGray, ColorReset)
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
			fmt.Printf("  %s•%s Supervisor daemon started successfully.\n", ColorGreen, ColorReset)
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
			fmt.Printf("  %s•%s %s\n", ColorYellow, ColorReset, res.Message)
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
	fmt.Printf("  %s•%s %sDaemon is not running.%s\n", ColorGray, ColorReset, ColorDim, ColorReset)
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

func FormatBytes(kb int64) string {
	b := kb * 1024
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func FormatUptime(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	d := seconds / 86400
	seconds %= 86400
	h := seconds / 3600
	seconds %= 3600
	m := seconds / 60
	s := seconds % 60

	var parts []string
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}
	return strings.Join(parts, " ")
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
	var dotColor string
	msgLower := strings.ToLower(res.Message)
	if strings.Contains(msgLower, "error") || strings.Contains(msgLower, "fail") {
		dotColor = ColorRed
	} else if strings.Contains(msgLower, "stop") || strings.Contains(msgLower, "purge") {
		dotColor = ColorYellow
	} else if strings.Contains(msgLower, "start") || strings.Contains(msgLower, "success") {
		dotColor = ColorGreen
	} else {
		dotColor = ColorCyan
	}

	if len(res.Processes) > 1 {
		for _, process := range res.Processes {
			fmt.Printf("  %s•%s %-10s %s%s%s %s(PID: %d)%s\n",
				dotColor, ColorReset, res.Message, ColorBold, process.Name, ColorReset, ColorGray, process.Pid, ColorReset)
		}
		return
	}
	if res.Pid > 0 {
		fmt.Printf("  %s•%s %-10s %s%s%s %s(PID: %d)%s\n",
			dotColor, ColorReset, res.Message, ColorBold, res.Name, ColorReset, ColorGray, res.Pid, ColorReset)
		return
	}
	if res.Name != "" {
		fmt.Printf("  %s•%s %-10s %s%s%s\n",
			dotColor, ColorReset, res.Message, ColorBold, res.Name, ColorReset)
		return
	}
	fmt.Printf("  %s•%s %s\n", dotColor, ColorReset, res.Message)
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
	fmt.Printf("  %s•%s %sStartup initialization configuration written and verified.%s\n", ColorGreen, ColorReset, ColorGray, ColorReset)
	return nil
}

func StartupUninstall() error {
	err := uninstallStartup()
	if err != nil {
		return err
	}
	fmt.Printf("  %s•%s %sStartup integration targets removed successfully.%s\n", ColorYellow, ColorReset, ColorGray, ColorReset)
	return nil
}

func Uninstall() error {
	fmt.Println()
	fmt.Printf("  %s%sCRITICAL WARNING: Supervisor environment purge request received.%s\n", ColorRed, ColorBold, ColorReset)
	fmt.Printf("  %s%s─────────────────────────────────────────────────────────────────%s\n", ColorDim, strings.Repeat("─", 33), ColorReset)
	fmt.Println("  This complete removal window executes the following actions:")
	fmt.Printf("    %s-%s Terminate background supervisor daemons safely\n", ColorRed, ColorReset)
	fmt.Printf("    %s-%s Erase platform startup hooks\n", ColorRed, ColorReset)
	fmt.Printf("    %s-%s Wipe local state directories, logging metrics, and caches\n", ColorRed, ColorReset)
	fmt.Printf("    %s-%s Remove installed zpm/zpmd executable files\n\n", ColorRed, ColorReset)

	fmt.Printf("  %sTo verify execution target parameters, type %syes%s%s to commit: ", ColorYellow, ColorBold, ColorReset, ColorYellow)
	fmt.Print(ColorReset)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "yes" {
		fmt.Printf("\n  %s•%s Action abandoned. Maintenance footprint unchanged.\n\n", ColorCyan, ColorReset)
		return nil
	}

	fmt.Printf("\n  %sProcessing Pipeline Tracks:%s\n", ColorBold, ColorReset)

	fmt.Printf("    %-45s", "Stopping persistent supervisor runner state...")
	stopDaemonByName(true)
	fmt.Printf("[%sDONE%s]\n", ColorGreen, ColorReset)

	fmt.Printf("    %-45s", "Dismantling global target run bindings...")
	err := uninstallStartup()
	if err != nil {
		fmt.Printf("[%sWARN: %v%s]\n", ColorYellow, err, ColorReset)
	} else {
		fmt.Printf("[%sDONE%s]\n", ColorGreen, ColorReset)
	}

	fmt.Printf("    %-45s", "Scrubbing system storage footprints...")
	dataDir := DataDir()
	err = os.RemoveAll(dataDir)
	if err != nil {
		fmt.Printf("[%sWARN: %v%s]\n", ColorYellow, err, ColorReset)
	} else {
		fmt.Printf("[%sDONE%s]\n", ColorGreen, ColorReset)
	}

	fmt.Printf("    %-45s", "Removing installed executable files...")
	err = uninstallInstallArtifacts()
	if err != nil {
		fmt.Printf("[%sWARN: %v%s]\n", ColorYellow, err, ColorReset)
	} else {
		fmt.Printf("[%sDONE%s]\n", ColorGreen, ColorReset)
	}

	fmt.Printf("\n  %s•%s Target environment registries clean.\n", ColorGreen, ColorReset)
	if note := uninstallInstallArtifactNote(); note != "" {
		fmt.Printf("  %sNote:%s %s\n", ColorYellow, ColorReset, note)
	}
	fmt.Println()

	return nil
}

func removeExistingPaths(paths ...string) error {
	seen := map[string]bool{}
	var errs []error
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		if seen[path] {
			continue
		}
		seen[path] = true

		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		if info.IsDir() {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

func knownExecutablePaths(names ...string) []string {
	paths := []string{}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, name := range names {
			paths = append(paths, filepath.Join(exeDir, name))
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range names {
			paths = append(paths, filepath.Join(home, ".local", "bin", name))
		}
	}
	return paths
}

func removeShellPathEntries(rcNames []string, markers []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var errs []error
	for _, rcName := range rcNames {
		path := filepath.Join(home, rcName)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		lines := strings.Split(string(data), "\n")
		kept := make([]string, 0, len(lines))
		changed := false
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			if line == "# ZPM (Zen Process Manager)" {
				if i+1 < len(lines) && containsAny(lines[i+1], markers) {
					i++
					changed = true
					continue
				}
			}
			if strings.HasPrefix(strings.TrimSpace(line), "export PATH=") && containsAny(line, markers) {
				changed = true
				continue
			}
			kept = append(kept, line)
		}
		if !changed {
			continue
		}
		err = os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0644)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func Usage() {
	fmt.Printf("\n  %sZPM %sProcess Supervisor Interface%s\n\n", ColorBold, ColorGray, ColorReset)
	fmt.Printf("  %sUsage:%s\n    zpp <command> [arguments]\n\n", ColorYellow, ColorReset)

	fmt.Printf("  %sCore Process Control Suite:%s\n", ColorCyan, ColorReset)
	fmt.Printf("    %-42s %sLaunch process inside manager lifecycle context%s\n", "start <cmd> [--name n] [--instances i]", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sRestore registered targeted app profiles%s\n", "start <app_name>", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sInstruct graceful instance drop signals%s\n", "stop <app_name>", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sForce rapid instance reset actions%s\n", "restart <app_name>", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sWipe specific context records and log dumps%s\n", "purge <app_name>", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sView live tabular diagnostic overview fields%s\n", "status | list", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sOpen animated responsive terminal dashboard%s\n\n", "tui | ui | tux", ColorDim, ColorReset)

	fmt.Printf("  %sDaemon Supervision & States:%s\n", ColorCyan, ColorReset)
	fmt.Printf("    %-42s %sModify daemon thread runtime states%s\n", "daemon start|stop|reload", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sManage supervisor initialization integrations%s\n\n", "startup install|uninstall", ColorDim, ColorReset)

	fmt.Printf("  %sEcosystem Workspace Targets:%s\n", ColorCyan, ColorReset)
	fmt.Printf("    %-42s %sBootstrap nested ecosystem definition matrices%s\n\n", "ecosystem start [config.json]", ColorDim, ColorReset)

	fmt.Printf("  %sRelease Publishing:%s\n", ColorCyan, ColorReset)
	fmt.Printf("    %-42s %sList GitHub release versions%s\n", "publish --list", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sBuild release archives locally%s\n", "publish build", ColorDim, ColorReset)
	fmt.Printf("    %-42s %sBuild, tag, release, and upload assets%s\n\n", "publish patch|minor|major", ColorDim, ColorReset)

	fmt.Printf("  %sSystem Maintenance Suites:%s\n", ColorCyan, ColorReset)
	fmt.Printf("    %-42s %sPurge supervisor structures from host profiles%s\n\n", "uninstall", ColorDim, ColorReset)
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
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
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
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
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
					fmt.Fprintf(os.Stderr, "%s[error]%s Missing value for environment parameter\n", ColorRed, ColorReset)
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
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
		for _, req := range requests {
			res, err := Send(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
				os.Exit(1)
			}
			PrintProcess(res)
		}
	case "start":
		if len(args) >= 2 && args[1] == "daemon" {
			err := StartDaemon()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
				os.Exit(1)
			}
			return
		}
		req, follow, err := ParseStart(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
		req.Cwd = cwd
		res, err := Send(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
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
				fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
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
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
		PrintProcess(res)
	case "list", "status":
		if len(args) > 1 && (args[1] == "--watch" || args[1] == "-w" || args[1] == "--tui") {
			err := RunTUI()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
				os.Exit(1)
			}
			return
		}
		res, err := Send(Request{Action: "status"})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
		PrintStatus(res.Processes)
	case "tui", "ui", "tux", "dashboard":
		err := RunTUI()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
	case "publish":
		err := publish.Run(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
	case "uninstall":
		err := Uninstall()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s[error]%s %v\n", ColorRed, ColorReset, err)
			os.Exit(1)
		}
	default:
		Usage()
		os.Exit(1)
	}
}
