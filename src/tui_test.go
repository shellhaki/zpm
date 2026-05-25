package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderStatusSnapshotResponsive(t *testing.T) {
	processes := sampleTUIProcesses()

	compact := RenderStatusSnapshot(processes, 56)
	if !strings.Contains(compact, "api") || !strings.Contains(compact, "worker") {
		t.Fatalf("compact snapshot omitted process names:\n%s", compact)
	}
	if strings.Contains(compact, "APP NAME") {
		t.Fatalf("compact snapshot rendered wide table header:\n%s", compact)
	}

	wide := RenderStatusSnapshot(processes, 120)
	if !strings.Contains(wide, "APP NAME") || !strings.Contains(wide, "COMMAND") {
		t.Fatalf("wide snapshot omitted table headings:\n%s", wide)
	}
}

func TestTUIViewStaysInsideCommonWidths(t *testing.T) {
	for _, width := range []int{48, 76, 120} {
		model := NewTUIModel()
		model.width = width
		model.height = 26
		model.processes = sampleTUIProcesses()
		model.lastUpdated = time.Now().Add(-time.Second)

		view := model.View()
		if got := maxLineWidth(view); got > width+6 {
			t.Fatalf("view width %d rendered too wide: got %d\n%s", width, got, view)
		}
	}
}

func sampleTUIProcesses() []Process {
	return []Process{
		{
			Name:          "api",
			Command:       "bun index.ts --port 3000",
			Cwd:           "/srv/zpm/api",
			Pid:           4242,
			Status:        "running",
			Healthy:       true,
			RestartCount:  2,
			MaxRestarts:   -1,
			MemoryRssKB:   24 * 1024,
			UptimeSeconds: 3661,
			LogPath:       "/tmp/api.log",
		},
		{
			Name:         "worker",
			Command:      "node worker.js --queue emails",
			Cwd:          "/srv/zpm/worker",
			Status:       "stopped",
			RestartCount: 1,
			MaxRestarts:  10,
			LogPath:      "/tmp/worker.log",
		},
	}
}

func maxLineWidth(value string) int {
	max := 0
	for _, line := range strings.Split(value, "\n") {
		if width := lipgloss.Width(line); width > max {
			max = width
		}
	}
	return max
}
