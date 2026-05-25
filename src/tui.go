package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tuiRefreshEvery = 2 * time.Second
	tuiFrameEvery   = 120 * time.Millisecond
)

var (
	tuiAccent  = lipgloss.Color("#47E5BC")
	tuiBlue    = lipgloss.Color("#74A7FF")
	tuiPink    = lipgloss.Color("#FF7AB6")
	tuiYellow  = lipgloss.Color("#FFD166")
	tuiRed     = lipgloss.Color("#FF5C7A")
	tuiGreen   = lipgloss.Color("#6EE7A8")
	tuiMuted   = lipgloss.Color("#7D8597")
	tuiPanel   = lipgloss.Color("#30384A")
	tuiSurface = lipgloss.Color("#151922")

	tuiTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(tuiAccent)
	tuiMutedStyle = lipgloss.NewStyle().
			Foreground(tuiMuted)
	tuiHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E9EEF7")).
			Background(tuiSurface).
			Padding(0, 1)
	tuiPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiPanel).
			Padding(0, 1)
	tuiSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#222B3D")).
				Foreground(lipgloss.Color("#F7FAFF"))
	tuiDangerStyle = lipgloss.NewStyle().
			Foreground(tuiRed).
			Bold(true)
)

type tuiStatusMsg struct {
	processes []Process
	err       error
	at        time.Time
}

type tuiTickMsg time.Time

type tuiModel struct {
	width       int
	height      int
	processes   []Process
	selected    int
	frame       int
	lastUpdated time.Time
	refreshing  bool
	err         error
}

type processSummary struct {
	total     int
	running   int
	stopped   int
	unhealthy int
	restarts  int
	memoryKB  int64
}

func NewTUIModel() tuiModel {
	return tuiModel{
		width:      TerminalWidth(),
		height:     28,
		refreshing: true,
	}
}

func RunTUI() error {
	if !stdoutIsTerminal() {
		res, err := Send(Request{Action: "status"})
		if err != nil {
			return err
		}
		fmt.Print(RenderStatusSnapshot(res.Processes, TerminalWidth()))
		return nil
	}

	_, err := tea.NewProgram(NewTUIModel(), tea.WithAltScreen()).Run()
	return err
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(tuiFetchStatus(), tuiNextFrame())
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "r":
			m.refreshing = true
			return m, tuiFetchStatus()
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.processes)-1 {
				m.selected++
			}
		case "home", "g":
			m.selected = 0
		case "end", "G":
			if len(m.processes) > 0 {
				m.selected = len(m.processes) - 1
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tuiStatusMsg:
		m.refreshing = false
		m.lastUpdated = msg.at
		m.err = msg.err
		if msg.err == nil {
			m.processes = msg.processes
			if m.selected >= len(m.processes) {
				m.selected = len(m.processes) - 1
			}
			if m.selected < 0 {
				m.selected = 0
			}
		}
	case tuiTickMsg:
		m.frame++
		cmds := []tea.Cmd{tuiNextFrame()}
		if !m.refreshing && (m.lastUpdated.IsZero() || time.Since(m.lastUpdated) >= tuiRefreshEvery) {
			m.refreshing = true
			cmds = append(cmds, tuiFetchStatus())
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m tuiModel) View() string {
	width := clampInt(m.width, 42, 220)
	height := clampInt(m.height, 10, 80)

	header := m.renderHeader(width)
	footer := m.renderFooter(width)
	if height <= 12 {
		return strings.Join([]string{
			header,
			m.renderTiny(width),
			footer,
		}, "\n")
	}

	parts := []string{header}
	if m.err != nil && len(m.processes) == 0 {
		parts = append(parts, m.renderError(width))
		parts = append(parts, footer)
		return strings.Join(parts, "\n")
	}

	summary := m.renderSummary(width)
	parts = append(parts, summary)

	used := lipgloss.Height(header) + lipgloss.Height(summary) + lipgloss.Height(footer) + 3
	available := maxInt(4, height-used)

	if width >= 118 && available >= 9 {
		leftWidth := width*2/3 - 1
		rightWidth := width - leftWidth - 2
		table := m.renderProcessList(leftWidth, available)
		details := m.renderDetails(rightWidth, available)
		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Top, table, "  ", details))
	} else {
		listRows := available
		if height >= 24 && len(m.processes) > 0 {
			listRows = maxInt(4, available-8)
		}
		parts = append(parts, m.renderProcessList(width, listRows))
		if height >= 24 && len(m.processes) > 0 {
			parts = append(parts, m.renderDetails(width, 7))
		}
	}

	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func tuiFetchStatus() tea.Cmd {
	return func() tea.Msg {
		res, err := Send(Request{Action: "status"})
		return tuiStatusMsg{processes: res.Processes, err: err, at: time.Now()}
	}
}

func tuiNextFrame() tea.Cmd {
	return tea.Tick(tuiFrameEvery, func(t time.Time) tea.Msg {
		return tuiTickMsg(t)
	})
}

func (m tuiModel) renderHeader(width int) string {
	spin := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinner := spin[m.frame%len(spin)]
	if !m.refreshing {
		spinner = "•"
	}

	summary := summarizeProcesses(m.processes)
	left := fmt.Sprintf("%s %s", tuiTitleStyle.Render("ZPM"), tuiMutedStyle.Render("live process deck"))
	right := tuiMutedStyle.Render(fmt.Sprintf("%s %d running / %d total", spinner, summary.running, summary.total))
	if m.err != nil {
		right = tuiDangerStyle.Render("daemon unreachable")
	}

	gap := maxInt(1, width-lipgloss.Width(left)-lipgloss.Width(right)-2)
	line := left + strings.Repeat(" ", gap) + right
	return tuiHeaderStyle.Width(width).Render(fit(line, width))
}

func (m tuiModel) renderSummary(width int) string {
	summary := summarizeProcesses(m.processes)
	cards := []string{
		renderMetricCard("RUNNING", strconv.Itoa(summary.running), "online", tuiGreen, cardWidth(width)),
		renderMetricCard("WATCHED", strconv.Itoa(summary.total), "tracked", tuiBlue, cardWidth(width)),
		renderMetricCard("MEMORY", FormatBytes(summary.memoryKB), "rss", tuiYellow, cardWidth(width)),
		renderMetricCard("RESTARTS", strconv.Itoa(summary.restarts), "lifetime", tuiPink, cardWidth(width)),
	}

	if width < 74 {
		return lipgloss.JoinVertical(lipgloss.Left, cards...)
	}
	if width < 118 {
		first := lipgloss.JoinHorizontal(lipgloss.Top, cards[0], " ", cards[1])
		second := lipgloss.JoinHorizontal(lipgloss.Top, cards[2], " ", cards[3])
		return lipgloss.JoinVertical(lipgloss.Left, first, second)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cards[0], " ", cards[1], " ", cards[2], " ", cards[3])
}

func (m tuiModel) renderProcessList(width int, rows int) string {
	if len(m.processes) == 0 {
		body := tuiMutedStyle.Render("No processes are being tracked yet.")
		hint := tuiMutedStyle.Render("Start one with zpm start \"bun index\" --name api")
		return panel(width).Render(body + "\n" + hint)
	}

	if width < 88 {
		return m.renderCompactList(width, rows)
	}
	return m.renderTable(width, rows)
}

func (m tuiModel) renderCompactList(width int, rows int) string {
	visible, offset := visibleProcessWindow(m.processes, m.selected, maxInt(1, rows))
	lines := []string{}
	for i, p := range visible {
		absolute := offset + i
		prefix := " "
		if absolute == m.selected {
			prefix = ">"
		}
		status := processBadge(p.Status, p.Healthy)
		meta := fmt.Sprintf("pid %s  %s  %s", pidText(p), memoryText(p), uptimeText(p))
		nameWidth := maxInt(8, width-lipgloss.Width(status)-lipgloss.Width(meta)-12)
		line := fmt.Sprintf("%s %s %s %s", prefix, pad(p.Name, nameWidth), status, tuiMutedStyle.Render(meta))
		line = fit(line, maxInt(20, width-5))
		if absolute == m.selected {
			line = tuiSelectedStyle.Width(maxInt(1, width-6)).Render(line)
		}
		lines = append(lines, line)
		if p.Command != "" && rows > 4 {
			cmd := "  " + tuiMutedStyle.Render(fit(p.Command, maxInt(10, width-8)))
			lines = append(lines, cmd)
		}
	}
	lines = appendOverflowHint(lines, len(m.processes), offset, len(visible))
	return panel(width).Render(strings.Join(lines, "\n"))
}

func (m tuiModel) renderTable(width int, rows int) string {
	visible, offset := visibleProcessWindow(m.processes, m.selected, maxInt(1, rows-2))
	inner := maxInt(60, width-6)
	nameW := clampInt(inner/5, 12, 22)
	statusW := 11
	pidW := 7
	memW := 9
	uptimeW := 11
	restartW := 8
	healthW := 8
	used := nameW + statusW + pidW + memW + uptimeW + restartW + healthW + 7
	cmdW := maxInt(12, inner-used)

	header := strings.Join([]string{
		pad("APP NAME", nameW),
		pad("STATUS", statusW),
		pad("PID", pidW),
		pad("MEM", memW),
		pad("UPTIME", uptimeW),
		pad("RESTART", restartW),
		pad("HEALTH", healthW),
		pad("COMMAND", cmdW),
	}, " ")

	lines := []string{tuiMutedStyle.Render(header)}
	for i, p := range visible {
		absolute := offset + i
		health := "--"
		if p.Status == "running" {
			if p.Healthy {
				health = "ok"
			} else {
				health = "bad"
			}
		}
		row := strings.Join([]string{
			pad(p.Name, nameW),
			pad(processBadge(p.Status, p.Healthy), statusW),
			pad(pidText(p), pidW),
			pad(memoryText(p), memW),
			pad(uptimeText(p), uptimeW),
			pad(strconv.Itoa(p.RestartCount), restartW),
			pad(health, healthW),
			pad(tuiMutedStyle.Render(p.Command), cmdW),
		}, " ")
		if absolute == m.selected {
			row = tuiSelectedStyle.Width(maxInt(1, width-6)).Render(fit(row, maxInt(1, width-6)))
		}
		lines = append(lines, row)
	}
	lines = appendOverflowHint(lines, len(m.processes), offset, len(visible))
	return panel(width).Render(strings.Join(lines, "\n"))
}

func (m tuiModel) renderDetails(width int, height int) string {
	if len(m.processes) == 0 {
		return panel(width).Render(tuiMutedStyle.Render("Select a process to inspect details."))
	}
	p := m.processes[clampInt(m.selected, 0, len(m.processes)-1)]
	lines := []string{
		tuiTitleStyle.Render(p.Name),
		fmt.Sprintf("%s %s", tuiMutedStyle.Render("status"), processBadge(p.Status, p.Healthy)),
		fmt.Sprintf("%s %s", tuiMutedStyle.Render("pid"), pidText(p)),
		fmt.Sprintf("%s %s", tuiMutedStyle.Render("cwd"), fit(p.Cwd, maxInt(12, width-10))),
		fmt.Sprintf("%s %s", tuiMutedStyle.Render("log"), fit(p.LogPath, maxInt(12, width-10))),
		fmt.Sprintf("%s %s", tuiMutedStyle.Render("cmd"), fit(p.Command, maxInt(12, width-10))),
		fmt.Sprintf("%s %d / %d", tuiMutedStyle.Render("restarts"), p.RestartCount, p.MaxRestarts),
	}
	if p.LastError != "" {
		lines = append(lines, fmt.Sprintf("%s %s", tuiMutedStyle.Render("health"), tuiDangerStyle.Render(fit(p.LastError, maxInt(12, width-12)))))
	}
	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return panel(width).Render(strings.Join(lines, "\n"))
}

func (m tuiModel) renderTiny(width int) string {
	summary := summarizeProcesses(m.processes)
	line := fmt.Sprintf("%d running / %d total", summary.running, summary.total)
	if m.err != nil {
		line = "daemon unreachable"
	}
	return panel(width).Render(fit(line, maxInt(10, width-4)))
}

func (m tuiModel) renderError(width int) string {
	lines := []string{
		tuiDangerStyle.Render("Could not reach the ZPM daemon."),
		tuiMutedStyle.Render(fit(m.err.Error(), maxInt(20, width-8))),
		tuiMutedStyle.Render("Run zpm daemon start, then reopen the dashboard."),
	}
	return panel(width).Render(strings.Join(lines, "\n"))
}

func (m tuiModel) renderFooter(width int) string {
	updated := "waiting for first refresh"
	if !m.lastUpdated.IsZero() {
		updated = "updated " + time.Since(m.lastUpdated).Truncate(time.Second).String() + " ago"
	}
	left := tuiMutedStyle.Render(updated)
	right := tuiMutedStyle.Render("r refresh  j/k move  q quit")
	gap := maxInt(1, width-lipgloss.Width(left)-lipgloss.Width(right)-2)
	return fit(left+strings.Repeat(" ", gap)+right, width)
}

func RenderStatusSnapshot(processes []Process, width int) string {
	width = clampInt(width, 48, 180)
	if len(processes) == 0 {
		return fmt.Sprintf("  %sNo active processes found under supervisor tracking.%s\n", ColorGray, ColorReset)
	}
	if width < 84 {
		return renderCompactStatusSnapshot(processes, width)
	}
	return renderWideStatusSnapshot(processes, width)
}

func renderCompactStatusSnapshot(processes []Process, width int) string {
	var out strings.Builder
	out.WriteString("\n")
	for _, p := range processes {
		status := ansiStatus(p)
		meta := fmt.Sprintf("pid %s  mem %s  up %s  restarts %d", pidText(p), memoryText(p), uptimeText(p), p.RestartCount)
		out.WriteString(fmt.Sprintf("  %s%s%s  %s\n", ColorBold, p.Name, ColorReset, status))
		out.WriteString(fmt.Sprintf("    %s%s%s\n", ColorDim, fit(meta, width-4), ColorReset))
		if p.Command != "" {
			out.WriteString(fmt.Sprintf("    %s%s%s\n", ColorDim, fit(p.Command, width-4), ColorReset))
		}
	}
	out.WriteString("\n")
	return out.String()
}

func renderWideStatusSnapshot(processes []Process, width int) string {
	inner := width - 4
	nameW := clampInt(inner/5, 14, 22)
	statusW := 10
	pidW := 8
	memW := 10
	uptimeW := 12
	restartW := 10
	healthW := 8
	used := nameW + statusW + pidW + memW + uptimeW + restartW + healthW + 7
	cmdW := maxInt(16, inner-used)

	var out strings.Builder
	out.WriteString("\n")
	out.WriteString(fmt.Sprintf("  %s%s%s\n", ColorDim, strings.Repeat("─", minInt(inner, 118)), ColorReset))
	out.WriteString(fmt.Sprintf("  %s%s%s\n", ColorBold, strings.Join([]string{
		pad("APP NAME", nameW),
		pad("STATUS", statusW),
		pad("PID", pidW),
		pad("MEMORY", memW),
		pad("UPTIME", uptimeW),
		pad("RESTARTS", restartW),
		pad("HEALTH", healthW),
		pad("COMMAND", cmdW),
	}, " "), ColorReset))
	out.WriteString(fmt.Sprintf("  %s%s%s\n", ColorDim, strings.Repeat("─", minInt(inner, 118)), ColorReset))

	for _, p := range processes {
		health := "--"
		if p.Status == "running" {
			if p.Healthy {
				health = ColorGreen + "ok" + ColorReset
			} else {
				health = ColorRed + "bad" + ColorReset
			}
		}
		out.WriteString("  ")
		out.WriteString(strings.Join([]string{
			pad(p.Name, nameW),
			pad(ansiStatus(p), statusW),
			pad(pidText(p), pidW),
			pad(memoryText(p), memW),
			pad(uptimeText(p), uptimeW),
			pad(strconv.Itoa(p.RestartCount), restartW),
			pad(health, healthW),
			ColorDim + pad(p.Command, cmdW) + ColorReset,
		}, " "))
		out.WriteString("\n")
	}
	out.WriteString("\n")
	return out.String()
}

func PrintStatus(processes []Process) {
	fmt.Print(RenderStatusSnapshot(processes, TerminalWidth()))
}

func summarizeProcesses(processes []Process) processSummary {
	summary := processSummary{total: len(processes)}
	for _, p := range processes {
		switch p.Status {
		case "running":
			summary.running++
			summary.memoryKB += p.MemoryRssKB
			if !p.Healthy {
				summary.unhealthy++
			}
		case "stopped", "crashed", "errored", "failed":
			summary.stopped++
		}
		summary.restarts += p.RestartCount
	}
	return summary
}

func renderMetricCard(label string, value string, caption string, color lipgloss.Color, width int) string {
	contentWidth := maxInt(10, width-4)
	valueStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	body := strings.Join([]string{
		tuiMutedStyle.Render(label),
		valueStyle.Render(fit(value, contentWidth)),
		tuiMutedStyle.Render(caption),
	}, "\n")
	return panel(width).Render(body)
}

func panel(width int) lipgloss.Style {
	return tuiPanelStyle.Width(maxInt(12, width-4))
}

func cardWidth(width int) int {
	switch {
	case width >= 118:
		return maxInt(22, (width-3)/4)
	case width >= 74:
		return maxInt(22, (width-1)/2)
	default:
		return width
	}
}

func processBadge(status string, healthy bool) string {
	label := strings.TrimSpace(status)
	if label == "" {
		label = "unknown"
	}
	color := tuiMuted
	switch label {
	case "running":
		if healthy {
			color = tuiGreen
		} else {
			color = tuiYellow
			label = "unhealthy"
		}
	case "stopped":
		color = tuiYellow
	case "restarting", "stopping":
		color = tuiBlue
	case "crashed", "errored", "failed":
		color = tuiRed
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(label)
}

func ansiStatus(p Process) string {
	switch p.Status {
	case "running":
		if p.Healthy {
			return ColorGreen + "running" + ColorReset
		}
		return ColorYellow + "unhealthy" + ColorReset
	case "stopped":
		return ColorYellow + "stopped" + ColorReset
	case "restarting", "stopping":
		return ColorCyan + p.Status + ColorReset
	case "crashed", "errored", "failed":
		return ColorRed + p.Status + ColorReset
	default:
		if p.Status == "" {
			return ColorGray + "unknown" + ColorReset
		}
		return p.Status
	}
}

func visibleProcessWindow(processes []Process, selected int, rows int) ([]Process, int) {
	if rows <= 0 || len(processes) == 0 {
		return nil, 0
	}
	if len(processes) <= rows {
		return processes, 0
	}
	selected = clampInt(selected, 0, len(processes)-1)
	start := selected - rows/2
	start = clampInt(start, 0, len(processes)-rows)
	return processes[start : start+rows], start
}

func appendOverflowHint(lines []string, total int, offset int, visible int) []string {
	if total <= visible {
		return lines
	}
	start := offset + 1
	end := offset + visible
	hint := fmt.Sprintf("showing %d-%d of %d", start, end, total)
	return append(lines, tuiMutedStyle.Render(hint))
}

func pidText(p Process) string {
	if p.Pid <= 0 {
		return "--"
	}
	return strconv.Itoa(p.Pid)
}

func memoryText(p Process) string {
	if p.Status != "running" {
		return "--"
	}
	return FormatBytes(p.MemoryRssKB)
}

func uptimeText(p Process) string {
	if p.Status != "running" {
		return "--"
	}
	return FormatUptime(p.UptimeSeconds)
}

func TerminalWidth() int {
	value := strings.TrimSpace(os.Getenv("COLUMNS"))
	if value == "" {
		return 100
	}
	width, err := strconv.Atoi(value)
	if err != nil || width <= 0 {
		return 100
	}
	return width
}

func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func fit(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	limit := maxInt(1, width-1)
	var out strings.Builder
	for _, r := range value {
		next := out.String() + string(r)
		if lipgloss.Width(next) > limit {
			break
		}
		out.WriteRune(r)
	}
	return out.String() + "…"
}

func pad(value string, width int) string {
	value = fit(value, width)
	padding := width - lipgloss.Width(value)
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
