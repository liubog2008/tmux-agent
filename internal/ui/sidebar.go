package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/liubog2008/tmux-agent/internal/state"
	"github.com/liubog2008/tmux-agent/internal/tmux"
)

type AgentItem struct {
	RuntimeKey  string
	Source      string
	SessionID   string
	SessionName string
	WindowID    string
	WindowName  string
	PaneID      string
	Status      string
	Title       string
	Repo        string
}

type SessionItem struct {
	SessionID     string
	SessionName   string
	WindowCount   string
	Attached      string
	SessionKind   string
	ContainsAgent bool
}

type snapshot struct {
	Agents   []AgentItem
	Sessions []SessionItem
	Err      error
}

type tickMsg time.Time

type snapshotMsg snapshot
type actionErrMsg struct {
	err error
}

type Model struct {
	agents        []AgentItem
	sessions      []SessionItem
	err           error
	width         int
	height        int
	activeSection string
	agentIndex    int
	sessionIndex  int
}

func NewModel() Model {
	return Model{activeSection: "agents"}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadSnapshotCmd(), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if pane := os.Getenv("TMUX_PANE"); pane != "" {
				return m, func() tea.Msg {
					_ = tmux.KillPane(pane)
					return nil
				}
			}
			return m, tea.Quit
		case "tab":
			if m.activeSection == "agents" {
				m.activeSection = "sessions"
			} else {
				m.activeSection = "agents"
			}
		case "j", "down":
			if m.activeSection == "agents" && m.agentIndex < len(m.agents)-1 {
				m.agentIndex++
			}
			if m.activeSection == "sessions" && m.sessionIndex < len(m.sessions)-1 {
				m.sessionIndex++
			}
		case "k", "up":
			if m.activeSection == "agents" && m.agentIndex > 0 {
				m.agentIndex--
			}
			if m.activeSection == "sessions" && m.sessionIndex > 0 {
				m.sessionIndex--
			}
		case "enter", "ctrl+m":
			if m.activeSection == "agents" && len(m.agents) > 0 {
				target := m.agents[m.agentIndex]
				return m, func() tea.Msg {
					if err := tmux.FocusPane(target.SessionID, target.WindowID, target.PaneID); err != nil {
						return actionErrMsg{err: err}
					}
					return nil
				}
			}
			if m.activeSection == "sessions" && len(m.sessions) > 0 {
				target := m.sessions[m.sessionIndex]
				return m, func() tea.Msg {
					if err := tmux.SwitchClient(target.SessionID); err != nil {
						return actionErrMsg{err: err}
					}
					return nil
				}
			}
		}
	case tickMsg:
		return m, tea.Batch(loadSnapshotCmd(), tickCmd())
	case snapshotMsg:
		m.agents = msg.Agents
		m.sessions = msg.Sessions
		if msg.Err != nil {
			m.err = msg.Err
		}
		if m.agentIndex >= len(m.agents) {
			m.agentIndex = max(0, len(m.agents)-1)
		}
		if m.sessionIndex >= len(m.sessions) {
			m.sessionIndex = max(0, len(m.sessions)-1)
		}
	case actionErrMsg:
		m.err = msg.err
	}
	return m, nil
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	activeTitleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	statusStyle := map[string]lipgloss.Style{
		state.StatusRunning:      lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		state.StatusWaitingInput: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		state.StatusError:        lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		state.StatusSuccess:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		state.StatusIdle:         lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		state.StatusStarting:     lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
	}

	var b strings.Builder
	agentTitle := titleStyle
	sessionTitle := titleStyle
	if m.activeSection == "agents" {
		agentTitle = activeTitleStyle
	} else {
		sessionTitle = activeTitleStyle
	}
	b.WriteString(agentTitle.Render("Agents"))
	b.WriteString("\n")
	if len(m.agents) == 0 {
		b.WriteString("  no active agent sessions\n")
	} else {
		for i, item := range m.agents {
			prefix := "  "
			if m.activeSection == "agents" && i == m.agentIndex {
				prefix = "> "
			}
			status := item.Status
			if status == "" {
				status = "unknown"
			}
			styler, ok := statusStyle[status]
			if !ok {
				styler = lipgloss.NewStyle()
			}
			label := item.Title
			if label == "" {
				label = item.RuntimeKey
			}
			line := fmt.Sprintf("%s%-7s %-18s %s", prefix, item.Source, truncate(label, 18), status)
			b.WriteString(styler.Render(line))
			b.WriteString("\n")
			if item.SessionName != "" {
				b.WriteString(fmt.Sprintf("    %s:%s\n", item.SessionName, item.PaneID))
			}
		}
	}
	b.WriteString("\n")
	b.WriteString(sessionTitle.Render("Sessions"))
	b.WriteString("\n")
	if len(m.sessions) == 0 {
		b.WriteString("  no normal sessions\n")
	} else {
		for i, item := range m.sessions {
			prefix := "  "
			if m.activeSection == "sessions" && i == m.sessionIndex {
				prefix = "> "
			}
			line := fmt.Sprintf("%s%-20s %s windows", prefix, truncate(item.SessionName, 20), item.WindowCount)
			b.WriteString(line)
			if item.Attached == "1" {
				b.WriteString(" attached")
			}
			b.WriteString("\n")
		}
	}
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.err.Error()))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString("tab switch  enter jump  q close")
	return b.String()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func loadSnapshotCmd() tea.Cmd {
	return func() tea.Msg {
		s, err := loadSnapshot()
		s.Err = err
		return snapshotMsg(s)
	}
}

func loadSnapshot() (snapshot, error) {
	var snap snapshot
	agentSessionName, err := tmux.ShowOption("@agent-sidebar-agent-session-name")
	if err != nil || agentSessionName == "" {
		agentSessionName = "__agent__"
	}

	panes, err := tmux.ListPanes()
	if err != nil {
		return snap, err
	}
	sessions, err := tmux.ListSessions()
	if err != nil {
		return snap, err
	}

	agentSessionSet := map[string]bool{}
	agentWindows := map[string]AgentItem{}
	for _, pane := range panes {
		if pane.SessionName != agentSessionName {
			continue
		}
		agentSessionSet[pane.SessionID] = true

		item := AgentItem{
			RuntimeKey:  pane.RuntimeKey,
			Source:      pane.AgentSource,
			SessionID:   pane.SessionID,
			SessionName: pane.SessionName,
			WindowID:    pane.WindowID,
			WindowName:  pane.WindowName,
			PaneID:      pane.PaneID,
			Status:      pane.AgentStatus,
			Title:       pane.AgentTitle,
			Repo:        pane.CurrentPath,
		}
		if item.Source == "" {
			item.Source = "agent"
		}
		if item.Title == "" {
			item.Title = pane.WindowName
		}
		if item.Status == "" {
			item.Status = state.StatusIdle
		}
		if pane.RuntimeKey != "" {
			if st, err := state.Load(pane.RuntimeKey); err == nil {
				if st.Title != "" {
					item.Title = st.Title
				}
				if st.Status != "" {
					item.Status = st.Status
				}
				if st.Source != "" {
					item.Source = st.Source
				}
				if st.Repo != "" {
					item.Repo = st.Repo
				}
			}
		}

		existing, ok := agentWindows[pane.WindowID]
		if !ok || preferAgentItem(item, existing) {
			agentWindows[pane.WindowID] = item
		}
	}

	for _, item := range agentWindows {
		snap.Agents = append(snap.Agents, item)
	}

	sort.Slice(snap.Agents, func(i, j int) bool {
		if snap.Agents[i].Source == snap.Agents[j].Source {
			return snap.Agents[i].Title < snap.Agents[j].Title
		}
		return snap.Agents[i].Source < snap.Agents[j].Source
	})

	for _, sess := range sessions {
		if sess.SessionName == agentSessionName {
			continue
		}
		containsAgent := agentSessionSet[sess.SessionID]
		kind := "normal"
		if containsAgent {
			kind = "agent"
		}
		if kind == "agent" || containsAgent {
			continue
		}
		snap.Sessions = append(snap.Sessions, SessionItem{
			SessionID:     sess.SessionID,
			SessionName:   sess.SessionName,
			WindowCount:   sess.WindowCount,
			Attached:      sess.Attached,
			SessionKind:   kind,
			ContainsAgent: containsAgent,
		})
	}
	sort.Slice(snap.Sessions, func(i, j int) bool {
		return snap.Sessions[i].SessionName < snap.Sessions[j].SessionName
	})
	return snap, nil
}

func truncate(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width < 2 {
		return s[:width]
	}
	return s[:width-1] + "…"
}

func preferAgentItem(candidate, existing AgentItem) bool {
	if existing.RuntimeKey == "" && candidate.RuntimeKey != "" {
		return true
	}
	if existing.RuntimeKey != "" && candidate.RuntimeKey == "" {
		return false
	}
	if candidate.PaneID < existing.PaneID {
		return true
	}
	return false
}
