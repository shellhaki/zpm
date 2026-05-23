package main

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

type Response struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Name      string    `json:"name"`
	Pid       int       `json:"pid"`
	Status    string    `json:"status"`
	LogPath   string    `json:"log_path"`
	Processes []Process `json:"processes"`
}
