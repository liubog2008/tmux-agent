package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Pane struct {
	SessionID      string
	SessionName    string
	WindowID       string
	WindowName     string
	PaneID         string
	CurrentPath    string
	Active         string
	AgentRole      string
	AgentSource    string
	RuntimeKey     string
	AgentStatus    string
	AgentUpdatedAt string
	AgentTitle     string
	SessionKind    string
}

type Session struct {
	SessionID    string
	SessionName  string
	WindowCount  string
	Attached     string
	LastAttached string
	SessionKind  string
}

func ListPanes() ([]Pane, error) {
	format := "#{session_id}|#{session_name}|#{window_id}|#{window_name}|#{pane_id}|#{pane_current_path}|#{pane_active}|#{@agent_role}|#{@agent_source}|#{@agent_runtime_key}|#{@agent_status}|#{@agent_updated_at}|#{@agent_title}|#{@session_kind}"
	out, err := run("list-panes", "-a", "-F", format)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	panes := make([]Pane, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 14 {
			continue
		}
		panes = append(panes, Pane{
			SessionID:      parts[0],
			SessionName:    parts[1],
			WindowID:       parts[2],
			WindowName:     parts[3],
			PaneID:         parts[4],
			CurrentPath:    parts[5],
			Active:         parts[6],
			AgentRole:      parts[7],
			AgentSource:    parts[8],
			RuntimeKey:     parts[9],
			AgentStatus:    parts[10],
			AgentUpdatedAt: parts[11],
			AgentTitle:     parts[12],
			SessionKind:    parts[13],
		})
	}
	return panes, nil
}

func ListSessions() ([]Session, error) {
	format := "#{session_id}|#{session_name}|#{session_windows}|#{session_attached}|#{session_last_attached}|#{@session_kind}"
	out, err := run("list-sessions", "-F", format)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	sessions := make([]Session, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}
		sessions = append(sessions, Session{
			SessionID:    parts[0],
			SessionName:  parts[1],
			WindowCount:  parts[2],
			Attached:     parts[3],
			LastAttached: parts[4],
			SessionKind:  parts[5],
		})
	}
	return sessions, nil
}

func SetPaneOption(paneID, name, value string) error {
	if paneID == "" {
		return fmt.Errorf("pane id is required")
	}
	_, err := run("set-option", "-p", "-t", paneID, name, value)
	return err
}

func SetSessionOption(sessionID, name, value string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	_, err := run("set-option", "-t", sessionID, name, value)
	return err
}

func SelectPane(paneID string) error {
	_, err := run("select-pane", "-t", paneID)
	return err
}

func SwitchClient(sessionID string) error {
	_, err := run("switch-client", "-t", sessionID)
	return err
}

func KillPane(paneID string) error {
	_, err := run("kill-pane", "-t", paneID)
	return err
}

func ShowOption(name string) (string, error) {
	out, err := run("show-option", "-gv", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func ShowPaneOption(paneID, name string) (string, error) {
	if paneID == "" {
		return "", fmt.Errorf("pane id is required")
	}
	out, err := run("show-option", "-p", "-v", "-t", paneID, name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func Format(format string) (string, error) {
	out, err := run("display-message", "-p", format)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func SetPaneTitle(paneID, title string) error {
	_, err := run("select-pane", "-t", paneID, "-T", title)
	return err
}

func SplitWindow(side, width string, command ...string) (string, error) {
	args := []string{"split-window", "-d", "-l", width, "-P", "-F", "#{pane_id}"}
	if side == "left" {
		args = append(args, "-hb")
	} else {
		args = append(args, "-h")
	}
	args = append(args, command...)
	out, err := run(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func NewWindow(path, name string, command ...string) (string, error) {
	args := []string{"new-window", "-P", "-F", "#{pane_id}"}
	if path != "" {
		args = append(args, "-c", path)
	}
	if name != "" {
		args = append(args, "-n", name)
	}
	args = append(args, command...)
	out, err := run(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func FindPaneByOption(name, value string) (string, error) {
	out, err := run("list-panes", "-F", fmt.Sprintf("#{pane_id}|#{%s}", name))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[1] == value {
			return parts[0], nil
		}
	}
	return "", nil
}

func run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("tmux %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}
