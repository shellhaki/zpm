package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	registry = make(map[string]Process)
	mutex    sync.Mutex
)

func DataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ".zpm"
	}
	return filepath.Join(dir, "zpm")
}

func LogsDir() string {
	return filepath.Join(DataDir(), "logs")
}

func RegistryPath() string {
	return filepath.Join(DataDir(), "registry.json")
}

func PidPath() string {
	return filepath.Join(DataDir(), "daemon.pid")
}

func LoadRegistry() error {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := os.ReadFile(RegistryPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &registry)
}

func SaveRegistryLocked() error {
	err := os.MkdirAll(DataDir(), 0755)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(RegistryPath(), data, 0644)
}

func GetProcess(name string) (Process, bool) {
	mutex.Lock()
	defer mutex.Unlock()
	process, ok := registry[name]
	return process, ok
}

func UpsertProcess(process Process) error {
	mutex.Lock()
	defer mutex.Unlock()
	registry[process.Name] = process
	return SaveRegistryLocked()
}

func UpdateProcess(name string, update func(Process) Process) error {
	mutex.Lock()
	defer mutex.Unlock()
	process, ok := registry[name]
	if !ok {
		return nil
	}
	registry[name] = update(process)
	return SaveRegistryLocked()
}

func SetProcessStopped(name string) error {
	mutex.Lock()
	defer mutex.Unlock()
	process, ok := registry[name]
	if !ok {
		return nil
	}
	process.Pid = 0
	process.Status = "stopped"
	process.StoppedAt = timeNow()
	registry[name] = process
	return SaveRegistryLocked()
}

func DeleteProcess(name string) error {
	mutex.Lock()
	defer mutex.Unlock()
	delete(registry, name)
	return SaveRegistryLocked()
}

func ListProcesses() []Process {
	mutex.Lock()
	defer mutex.Unlock()
	processes := make([]Process, 0, len(registry))
	for _, process := range registry {
		processes = append(processes, process)
	}
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Name < processes[j].Name
	})
	return processes
}

func ResolveTargets(name string) ([]Process, error) {
	mutex.Lock()
	defer mutex.Unlock()
	name = CleanName(name)
	if name == "" {
		return nil, errors.New("process name required")
	}
	targets := []Process{}
	if process, ok := registry[name]; ok {
		targets = append(targets, process)
	}
	prefix := name + "-"
	for processName, process := range registry {
		if strings.HasPrefix(processName, prefix) {
			targets = append(targets, process)
		}
	}
	if len(targets) == 0 {
		return nil, errors.New("process not found")
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Name < targets[j].Name
	})
	return targets, nil
}

func ResolveName(name string) (Process, error) {
	mutex.Lock()
	defer mutex.Unlock()
	if name != "" {
		process, ok := registry[name]
		if !ok {
			return Process{}, errors.New("process not found")
		}
		return process, nil
	}
	if len(registry) != 1 {
		return Process{}, errors.New("process name required")
	}
	for _, process := range registry {
		return process, nil
	}
	return Process{}, errors.New("process not found")
}
