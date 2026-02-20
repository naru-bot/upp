package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/naru-bot/watchdog/internal/checker"
	"github.com/naru-bot/watchdog/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "tui",
		Short: "Interactive terminal dashboard for managing monitors",
		Long: `Launch an interactive TUI dashboard.

Navigate with arrow keys, manage targets, view details, and run checks
all from a single terminal interface. Works great over SSH.

Keybindings:
  ↑/↓/j/k  Navigate targets
  Enter     View target details
  c         Run check on selected target
  C         Run check on all targets
  a         Add new target
  d         Delete selected target
  p         Pause/unpause selected target
  r         Refresh dashboard
  ?         Toggle help
  q/Esc     Quit`,
		Run: func(cmd *cobra.Command, args []string) {
			p := tea.NewProgram(newTUIModel(), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				exitError(err.Error())
			}
		},
	})
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#353533")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	upStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	downStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4672")).
			Bold(true)

	changedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFBF00")).
			Bold(true)

	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Width(60)

	sparkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))
)

type view int

const (
	viewList view = iota
	viewDetail
	viewEdit
	viewAdd
)

// Edit field indices
const (
	editName = iota
	editURL
	editType
	editInterval
	editTimeout
	editRetries
	editSelector
	editExpected
	editThreshold
	editFieldCount
)

var editFieldLabels = [editFieldCount]string{
	"Name", "URL", "Type", "Interval (s)", "Timeout (s)", "Retries", "Selector", "Expect", "Threshold (%)",
}

var typeOptions = []string{"http", "tcp", "ping", "dns", "visual"}

func nextType(current string) string {
	for i, t := range typeOptions {
		if t == current {
			return typeOptions[(i+1)%len(typeOptions)]
		}
	}
	return typeOptions[0]
}

func prevType(current string) string {
	for i, t := range typeOptions {
		if t == current {
			return typeOptions[(i-1+len(typeOptions))%len(typeOptions)]
		}
	}
	return typeOptions[0]
}

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Check   key.Binding
	CheckAll key.Binding
	Delete  key.Binding
	Pause   key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
	Back    key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
	Check:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "check")),
	CheckAll: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "check all")),
	Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Pause:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause/resume")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
	Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Check, k.CheckAll, k.Pause, k.Delete, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Check, k.CheckAll, k.Refresh},
		{k.Pause, k.Delete},
		{k.Help, k.Quit},
	}
}

type tickMsg time.Time
type checkDoneMsg struct {
	targetID int64
	result   *checker.Result
}

type tuiModel struct {
	table      table.Model
	targets    []db.Target
	results    map[int64]*checker.Result
	view       view
	selected   *db.Target
	detail     string
	help       help.Model
	showHelp   bool
	status     string
	width      int
	height     int
	checking   bool
	editInputs []textinput.Model
	editFocus  int
}

func newTUIModel() tuiModel {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Name", Width: 20},
		{Title: "URL", Width: 35},
		{Title: "Type", Width: 6},
		{Title: "Status", Width: 12},
		{Title: "Response", Width: 10},
		{Title: "Uptime", Width: 8},
		{Title: "Trend", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(false)
	t.SetStyles(s)

	return tuiModel{
		table:   t,
		results: make(map[int64]*checker.Result),
		help:    help.New(),
		status:  "Loading...",
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadTargets,
		m.tick(),
	)
}

func (m tuiModel) tick() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m tuiModel) loadTargets() tea.Msg {
	return tickMsg(time.Now())
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 8)
		m.help.Width = msg.Width

	case tea.KeyMsg:
		if m.view == viewEdit {
			switch msg.Type {
			case tea.KeyEsc:
				m.view = viewDetail
				m.updateDetail()
				return m, nil
			case tea.KeyTab, tea.KeyDown:
				m.editFocus = (m.editFocus + 1) % editFieldCount
				return m, m.focusEditField()
			case tea.KeyShiftTab, tea.KeyUp:
				m.editFocus = (m.editFocus - 1 + editFieldCount) % editFieldCount
				return m, m.focusEditField()
			case tea.KeyEnter:
				if err := m.saveEdit(); err != nil {
					m.status = fmt.Sprintf("Error: %s", err)
				} else {
					m.status = fmt.Sprintf("✓ Updated: %s", m.selected.Name)
					m.refreshData()
					m.updateDetail()
				}
				m.view = viewDetail
				return m, nil
			case tea.KeyLeft, tea.KeyRight, tea.KeySpace:
				// Type field: cycle through options
				if m.editFocus == editType {
					cur := m.editInputs[editType].Value()
					if msg.Type == tea.KeyLeft {
						m.editInputs[editType].SetValue(prevType(cur))
					} else {
						m.editInputs[editType].SetValue(nextType(cur))
					}
					return m, nil
				}
			}
			// Skip text input update for type field (not freeform)
			if m.editFocus == editType {
				// Only allow tab/arrow navigation, ignore typed chars
				if msg.Type == tea.KeyRunes {
					return m, nil
				}
			}
			// Update the focused input
			var cmd tea.Cmd
			m.editInputs[m.editFocus], cmd = m.editInputs[m.editFocus].Update(msg)
			return m, cmd
		}

		if m.view == viewAdd {
			switch msg.Type {
			case tea.KeyEsc:
				m.view = viewList
				return m, nil
			case tea.KeyTab, tea.KeyDown:
				m.editFocus = (m.editFocus + 1) % editFieldCount
				return m, m.focusEditField()
			case tea.KeyShiftTab, tea.KeyUp:
				m.editFocus = (m.editFocus - 1 + editFieldCount) % editFieldCount
				return m, m.focusEditField()
			case tea.KeyEnter:
				if err := m.saveAdd(); err != nil {
					m.status = fmt.Sprintf("Error: %s", err)
				} else {
					m.status = "✓ Target added"
					m.refreshData()
				}
				m.view = viewList
				return m, nil
			case tea.KeyLeft, tea.KeyRight, tea.KeySpace:
				if m.editFocus == editType {
					cur := m.editInputs[editType].Value()
					if msg.Type == tea.KeyLeft {
						m.editInputs[editType].SetValue(prevType(cur))
					} else {
						m.editInputs[editType].SetValue(nextType(cur))
					}
					return m, nil
				}
			}
			if m.editFocus == editType {
				if msg.Type == tea.KeyRunes {
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.editInputs[m.editFocus], cmd = m.editInputs[m.editFocus].Update(msg)
			return m, cmd
		}

		if m.view == viewDetail {
			switch {
			case key.Matches(msg, keys.Quit), key.Matches(msg, keys.Back):
				m.view = viewList
				m.selected = nil
				return m, nil
			case key.Matches(msg, keys.Check):
				if m.selected != nil {
					m.status = fmt.Sprintf("Checking %s...", m.selected.Name)
					return m, m.runCheck(m.selected)
				}
			case msg.String() == "e":
				if m.selected != nil {
					m.initEditInputs()
					m.view = viewEdit
					return m, m.focusEditField()
				}
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, keys.Enter):
			row := m.table.Cursor()
			if row >= 0 && row < len(m.targets) {
				m.selected = &m.targets[row]
				m.view = viewDetail
				m.updateDetail()
			}
			return m, nil
		case key.Matches(msg, keys.Check):
			return m, m.checkSelected()
		case key.Matches(msg, keys.CheckAll):
			m.status = "Checking all targets..."
			m.checking = true
			var cmds []tea.Cmd
			for i := range m.targets {
				cmds = append(cmds, m.runCheck(&m.targets[i]))
			}
			return m, tea.Batch(cmds...)
		case key.Matches(msg, keys.Delete):
			return m, m.deleteSelected()
		case key.Matches(msg, keys.Pause):
			return m, m.togglePause()
		case key.Matches(msg, keys.Refresh):
			m.status = "Refreshing..."
			return m, m.loadTargets
		case msg.String() == "a":
			m.initAddInputs()
			m.view = viewAdd
			return m, m.focusEditField()
		}

	case tickMsg:
		m.refreshData()
		m.status = fmt.Sprintf("Last refresh: %s | %d targets", time.Now().Format("15:04:05"), len(m.targets))
		return m, m.tick()

	case checkDoneMsg:
		m.results[msg.targetID] = msg.result
		// Save result to DB
		cr := &db.CheckResult{
			TargetID:     msg.targetID,
			Status:       msg.result.Status,
			StatusCode:   msg.result.StatusCode,
			ResponseTime: msg.result.ResponseTime.Milliseconds(),
			ContentHash:  msg.result.ContentHash,
			Error:        msg.result.Error,
		}
		db.SaveCheckResult(cr)
		if msg.result.Content != "" && msg.result.ContentHash != "" {
			snaps, _ := db.GetLatestSnapshots(msg.targetID, 1)
			if len(snaps) == 0 || snaps[0].Hash != msg.result.ContentHash {
				db.SaveSnapshot(msg.targetID, msg.result.Content, msg.result.ContentHash)
			}
		}
		m.refreshData()
		m.status = fmt.Sprintf("Checked | %d targets | %s", len(m.targets), time.Now().Format("15:04:05"))
		if m.view == viewDetail && m.selected != nil && m.selected.ID == msg.targetID {
			m.updateDetail()
		}
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *tuiModel) refreshData() {
	targets, err := db.ListTargets()
	if err != nil {
		return
	}
	m.targets = targets

	var rows []table.Row
	for _, t := range targets {
		status := "—"
		respTime := "—"
		uptime := "—"
		spark := ""

		results, _ := db.GetCheckHistory(t.ID, 20)
		if len(results) > 0 {
			last := results[0]
			status = last.Status
			respTime = fmt.Sprintf("%dms", last.ResponseTime)

			// Calculate uptime
			since := time.Now().Add(-24 * time.Hour)
			total, up, _, _ := db.GetUptimeStats(t.ID, since)
			if total > 0 {
				pct := float64(up) / float64(total) * 100
				uptime = fmt.Sprintf("%.1f%%", pct)
			}

			// Build sparkline
			var times []int64
			for _, r := range results {
				times = append(times, r.ResponseTime)
			}
			spark = buildSparkline(times, 10)
		}

		if t.Paused {
			status = "paused"
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", t.ID),
			t.Name,
			truncate(t.URL, 35),
			t.Type,
			status,
			respTime,
			uptime,
			spark,
		})
	}
	m.table.SetRows(rows)
}

func (m *tuiModel) updateDetail() {
	if m.selected == nil {
		return
	}
	t := m.selected
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Name:     %s\n", t.Name))
	sb.WriteString(fmt.Sprintf("URL:      %s\n", t.URL))
	sb.WriteString(fmt.Sprintf("Type:     %s\n", t.Type))
	sb.WriteString(fmt.Sprintf("Interval: %ds\n", t.Interval))
	if t.Selector != "" {
		sb.WriteString(fmt.Sprintf("Selector: %s\n", t.Selector))
	}
	if t.Expect != "" {
		sb.WriteString(fmt.Sprintf("Expect:   %s\n", t.Expect))
	}
	sb.WriteString(fmt.Sprintf("Timeout:  %ds | Retries: %d\n", t.Timeout, t.Retries))
	sb.WriteString(fmt.Sprintf("Paused:   %v\n", t.Paused))
	sb.WriteString("\n")

	// Last error
	lastResults, _ := db.GetCheckHistory(t.ID, 1)
	if len(lastResults) > 0 && lastResults[0].Error != "" {
		sb.WriteString(fmt.Sprintf("⚠ Error:  %s\n", lastResults[0].Error))
	}
	sb.WriteString("\n")

	// Stats
	since := time.Now().Add(-24 * time.Hour)
	total, up, avgMs, _ := db.GetUptimeStats(t.ID, since)
	if total > 0 {
		pct := float64(up) / float64(total) * 100
		sb.WriteString(fmt.Sprintf("Uptime (24h):  %.2f%% (%d/%d checks)\n", pct, up, total))
		sb.WriteString(fmt.Sprintf("Avg Response:  %.0fms\n", avgMs))
	}

	// Recent history
	results, _ := db.GetCheckHistory(t.ID, 10)
	if len(results) > 0 {
		sb.WriteString("\nRecent checks:\n")
		for _, r := range results {
			icon := "●"
			switch r.Status {
			case "up", "unchanged":
				icon = "✓"
			case "changed":
				icon = "△"
			case "down", "error":
				icon = "✗"
			}
			line := fmt.Sprintf("  %s  %s  %dms  %s", r.CheckedAt.Format("15:04:05"), icon, r.ResponseTime, r.Status)
			if r.Error != "" {
				line += fmt.Sprintf(" — %s", r.Error)
			}
			sb.WriteString(line + "\n")
		}

		// Sparkline
		var times []int64
		for _, r := range results {
			times = append(times, r.ResponseTime)
		}
		spark := buildSparkline(times, 20)
		sb.WriteString(fmt.Sprintf("\nResponse trend: %s\n", spark))
	}

	m.detail = sb.String()
}

func (m *tuiModel) initAddInputs() {
	m.editInputs = make([]textinput.Model, editFieldCount)

	for i := 0; i < editFieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = fmt.Sprintf("  %s: ", editFieldLabels[i])
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		ti.CharLimit = 256
		m.editInputs[i] = ti
	}

	m.editInputs[editName].Placeholder = "My Site"
	m.editInputs[editURL].Placeholder = "https://example.com"
	m.editInputs[editType].SetValue("http")
	m.editInputs[editInterval].SetValue("300")
	m.editInputs[editTimeout].SetValue("30")
	m.editInputs[editRetries].SetValue("1")
	m.editInputs[editSelector].Placeholder = "CSS selector (optional)"
	m.editInputs[editExpected].Placeholder = "Expected keyword (optional)"
	m.editInputs[editThreshold].SetValue("5.0")

	m.editFocus = 0
}

func (m *tuiModel) saveAdd() error {
	url := m.editInputs[editURL].Value()
	if url == "" {
		return fmt.Errorf("URL is required")
	}

	name := m.editInputs[editName].Value()
	typ := m.editInputs[editType].Value()
	selector := m.editInputs[editSelector].Value()
	expect := m.editInputs[editExpected].Value()

	interval := 300
	if v, err := strconv.Atoi(m.editInputs[editInterval].Value()); err == nil && v > 0 {
		interval = v
	}
	timeout := 30
	if v, err := strconv.Atoi(m.editInputs[editTimeout].Value()); err == nil && v > 0 {
		timeout = v
	}
	retries := 1
	if v, err := strconv.Atoi(m.editInputs[editRetries].Value()); err == nil && v > 0 {
		retries = v
	}
	threshold := 5.0
	if v, err := strconv.ParseFloat(m.editInputs[editThreshold].Value(), 64); err == nil && v > 0 {
		threshold = v
	}

	_, err := db.AddTarget(name, url, typ, interval, selector, "", expect, timeout, retries, threshold)
	return err
}

func (m *tuiModel) initEditInputs() {
	m.editInputs = make([]textinput.Model, editFieldCount)
	t := m.selected

	for i := 0; i < editFieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = fmt.Sprintf("  %s: ", editFieldLabels[i])
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		ti.CharLimit = 256
		m.editInputs[i] = ti
	}

	m.editInputs[editName].SetValue(t.Name)
	m.editInputs[editURL].SetValue(t.URL)
	m.editInputs[editType].SetValue(t.Type)
	m.editInputs[editType].Placeholder = "http, tcp, ping, dns, visual"
	m.editInputs[editInterval].SetValue(fmt.Sprintf("%d", t.Interval))
	m.editInputs[editTimeout].SetValue(fmt.Sprintf("%d", t.Timeout))
	m.editInputs[editRetries].SetValue(fmt.Sprintf("%d", t.Retries))
	m.editInputs[editSelector].SetValue(t.Selector)
	m.editInputs[editSelector].Placeholder = "CSS selector (optional)"
	m.editInputs[editExpected].SetValue(t.Expect)
	m.editInputs[editExpected].Placeholder = "Expected keyword (optional)"
	m.editInputs[editThreshold].SetValue(fmt.Sprintf("%.1f", t.Threshold))

	m.editFocus = 0
}

func (m *tuiModel) focusEditField() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.editInputs {
		if i == m.editFocus {
			cmds = append(cmds, m.editInputs[i].Focus())
			m.editInputs[i].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
		} else {
			m.editInputs[i].Blur()
			m.editInputs[i].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
		}
	}
	return tea.Batch(cmds...)
}

func (m *tuiModel) saveEdit() error {
	t := m.selected
	t.Name = m.editInputs[editName].Value()
	t.URL = m.editInputs[editURL].Value()
	t.Type = m.editInputs[editType].Value()
	t.Selector = m.editInputs[editSelector].Value()
	t.Expect = m.editInputs[editExpected].Value()

	if v, err := strconv.Atoi(m.editInputs[editInterval].Value()); err == nil && v > 0 {
		t.Interval = v
	}
	if v, err := strconv.Atoi(m.editInputs[editTimeout].Value()); err == nil && v > 0 {
		t.Timeout = v
	}
	if v, err := strconv.Atoi(m.editInputs[editRetries].Value()); err == nil && v > 0 {
		t.Retries = v
	}
	if v, err := strconv.ParseFloat(m.editInputs[editThreshold].Value(), 64); err == nil && v > 0 {
		t.Threshold = v
	}

	return db.UpdateTarget(t)
}

func (m *tuiModel) runCheck(t *db.Target) tea.Cmd {
	target := *t
	return func() tea.Msg {
		result := checker.Check(&target)
		return checkDoneMsg{targetID: target.ID, result: result}
	}
}

func (m *tuiModel) checkSelected() tea.Cmd {
	row := m.table.Cursor()
	if row >= 0 && row < len(m.targets) {
		t := &m.targets[row]
		m.status = fmt.Sprintf("Checking %s...", t.Name)
		return m.runCheck(t)
	}
	return nil
}

func (m *tuiModel) deleteSelected() tea.Cmd {
	row := m.table.Cursor()
	if row >= 0 && row < len(m.targets) {
		t := m.targets[row]
		db.RemoveTarget(fmt.Sprintf("%d", t.ID))
		m.status = fmt.Sprintf("Deleted: %s", t.Name)
		m.refreshData()
	}
	return nil
}

func (m *tuiModel) togglePause() tea.Cmd {
	row := m.table.Cursor()
	if row >= 0 && row < len(m.targets) {
		t := m.targets[row]
		db.SetPaused(fmt.Sprintf("%d", t.ID), !t.Paused)
		action := "Paused"
		if t.Paused {
			action = "Resumed"
		}
		m.status = fmt.Sprintf("%s: %s", action, t.Name)
		m.refreshData()
	}
	return nil
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Title bar
	title := titleStyle.Render(" 🐕 Watchdog ")
	sb.WriteString(title + "\n\n")

	if m.view == viewAdd || (m.view == viewEdit && m.selected != nil) {
		// Edit view
		headerText := "Add New Target"
		if m.view == viewEdit && m.selected != nil {
			headerText = fmt.Sprintf("Editing: %s", m.selected.Name)
		}
		header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render(headerText)
		sb.WriteString(header + "\n\n")

		var fields strings.Builder
		for i := 0; i < editFieldCount; i++ {
			cursor := "  "
			if i == m.editFocus {
				cursor = "▸ "
			}
			if i == editType {
				// Render as selectable options instead of text input
				label := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render(
					fmt.Sprintf("  %s: ", editFieldLabels[i]))
				cur := m.editInputs[editType].Value()
				var opts []string
				for _, t := range typeOptions {
					if t == cur {
						opts = append(opts, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Render("["+t+"]"))
					} else {
						opts = append(opts, lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(" "+t+" "))
					}
				}
				fields.WriteString(cursor + label + strings.Join(opts, " ") + "\n")
			} else {
				fields.WriteString(cursor + m.editInputs[i].View() + "\n")
			}
		}
		sb.WriteString(detailBoxStyle.Width(70).Render(fields.String()))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("↑↓/tab: navigate • ←→/space: change type • enter: save • esc: cancel"))
	} else if m.view == viewDetail && m.selected != nil {
		// Detail view
		header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render(
			fmt.Sprintf("Target: %s", m.selected.Name))
		sb.WriteString(header + "\n\n")
		sb.WriteString(detailBoxStyle.Render(m.detail))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("e: edit • c: check • esc: back"))
	} else {
		// List view
		sb.WriteString(m.table.View())
		sb.WriteString("\n\n")

		// Status bar
		sb.WriteString(statusBarStyle.Render(m.status))
		sb.WriteString("\n")

		// Help
		if m.showHelp {
			sb.WriteString("\n" + m.help.View(keys))
		} else {
			sb.WriteString(helpStyle.Render("\n ↑↓ navigate • enter details • c check • C check all • p pause • d delete • ? help • q quit"))
		}
	}

	return sb.String()
}
