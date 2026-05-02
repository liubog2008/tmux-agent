package app

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liubog2008/tmux-agent/internal/state"
	"github.com/liubog2008/tmux-agent/internal/tmux"
	"github.com/liubog2008/tmux-agent/internal/ui"
)

func Run(args []string) error {
	if len(args) < 2 {
		usage(args[0])
		return errors.New("subcommand is required")
	}

	var err error
	switch args[1] {
	case "sidebar":
		err = runSidebar()
	case "status-segment":
		err = runStatusSegment()
	case "toggle":
		err = runToggle()
	case "open":
		err = runOpen()
	case "close":
		err = runClose(args[2:])
	case "switch-window":
		err = runSwitchWindow()
	case "prepare":
		err = runPrepare(args[2:])
	case "start":
		err = runStart(args[2:])
	case "update":
		err = runUpdate(args[2:])
	case "finish":
		err = runFinish(args[2:])
	case "cleanup":
		err = runCleanup(args[2:])
	case "help":
		usage(args[0])
		return nil
	default:
		usage(args[0])
		return fmt.Errorf("unknown subcommand: %s", args[1])
	}

	return err
}

func runSidebar() error {
	program := tea.NewProgram(ui.NewModel())
	_, err := program.Run()
	return err
}

func runToggle() error {
	paneID, err := tmux.FindPaneByOption("@agent_sidebar_role", "sidebar")
	if err != nil {
		return err
	}
	if paneID != "" {
		return tmux.KillPane(paneID)
	}
	return runOpen()
}

func runOpen() error {
	width, err := tmux.ShowOption("@agent-sidebar-width")
	if err != nil || width == "" {
		width = "42"
	}
	side, err := tmux.ShowOption("@agent-sidebar-side")
	if err != nil || side == "" {
		side = "right"
	}

	exe, err := selfExecutable()
	if err != nil {
		return err
	}

	paneID, err := tmux.SplitWindow(side, width, exe, "sidebar")
	if err != nil {
		return err
	}
	if err := tmux.SetPaneOption(paneID, "@agent_sidebar_role", "sidebar"); err != nil {
		return err
	}
	if err := tmux.SetPaneTitle(paneID, "tmux-agent-sidebar"); err != nil {
		return err
	}
	return tmux.SelectPane(paneID)
}

func runClose(args []string) error {
	fs := flag.NewFlagSet("close", flag.ContinueOnError)
	paneID := fs.String("pane-id", "", "sidebar pane id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	targetPane := *paneID
	if targetPane != "" {
		role, err := tmux.ShowPaneOption(targetPane, "@agent_sidebar_role")
		if err != nil {
			return err
		}
		if role != "sidebar" {
			return nil
		}
	} else {
		var err error
		targetPane, err = tmux.FindPaneByOption("@agent_sidebar_role", "sidebar")
		if err != nil {
			return err
		}
	}
	if targetPane == "" {
		return nil
	}
	return tmux.KillPane(targetPane)
}

func runSwitchWindow() error {
	currentPaneID, _ := tmux.Format("#{pane_id}")
	currentSessionID, _ := tmux.Format("#{session_id}")
	currentSessionName, _ := tmux.Format("#{session_name}")
	currentWindowID, _ := tmux.Format("#{window_id}")
	currentWindowName, _ := tmux.Format("#{window_name}")

	currentPath, err := tmux.Format("#{pane_current_path}")
	if err != nil || currentPath == "" {
		currentPath, _ = os.Getwd()
	}

	agentSessionName, err := tmux.ShowOption("@agent-sidebar-agent-session-name")
	if err != nil || agentSessionName == "" {
		agentSessionName = "__agent__"
	}

	panes, err := tmux.ListPanes()
	if err != nil {
		return err
	}

	currentPane := findPaneByID(panes, currentPaneID)
	currentIsAgent := currentSessionName == agentSessionName
	if currentPane != nil && currentPane.WindowKind == "agent" {
		currentIsAgent = true
	}

	if currentIsAgent {
		targetPane, targetSessionTarget, targetWindowID, targetWindowName := resolveNormalWindowTarget(currentPane)
		if targetPane != "" {
			return tmux.FocusPane(targetSessionTarget, targetWindowID, targetPane)
		}
		if targetSessionTarget == "" {
			return errors.New("agent window is missing owner session metadata")
		}
		if targetWindowName == "" {
			targetWindowName = "main"
		}
		paneID, err := tmux.NewWindowInSession(targetSessionTarget, currentPath, targetWindowName)
		if err != nil {
			return err
		}
		return tmux.FocusPane(targetSessionTarget, "", paneID)
	}

	for _, pane := range panes {
		if pane.SessionName != agentSessionName {
			continue
		}
		if pane.OwnerSessionID == currentSessionID && pane.OwnerWindowID == currentWindowID {
			return tmux.FocusPane(pane.SessionID, pane.WindowID, pane.PaneID)
		}
	}

	return createAgentWindow(currentPath, currentSessionID, currentSessionName, currentWindowID, currentWindowName)
}

func createAgentWindow(currentPath, ownerSessionID, ownerSessionName, ownerWindowID, ownerWindowName string) error {
	exe, err := selfExecutable()
	if err != nil {
		return err
	}

	windowName, err := tmux.ShowOption("@agent-sidebar-agent-window-name")
	if err != nil || windowName == "" {
		windowName = "agent"
	}
	agentSessionName, err := tmux.ShowOption("@agent-sidebar-agent-session-name")
	if err != nil || agentSessionName == "" {
		agentSessionName = "__agent__"
	}

	defaultShell, err := tmux.ShowOption("default-shell")
	if err != nil || defaultShell == "" {
		defaultShell = os.Getenv("SHELL")
	}
	if defaultShell == "" {
		defaultShell = "/bin/sh"
	}

	runtimeKey := fmt.Sprintf("agent-%d-%d", time.Now().Unix(), os.Getpid())
	sessionExists, err := tmux.HasSession(agentSessionName)
	if err != nil {
		return err
	}

	command := []string{
		"env",
		"TMUX_AGENT_RUNTIME_KEY=" + runtimeKey,
		"TMUX_AGENT_BIN=" + exe,
		defaultShell,
		"-l",
	}

	var paneID string
	if sessionExists {
		paneID, err = tmux.NewWindowInSession(
			agentSessionName,
			currentPath,
			windowName,
			command...,
		)
	} else {
		paneID, err = tmux.NewDetachedSession(
			agentSessionName,
			currentPath,
			windowName,
			command...,
		)
	}
	if err != nil {
		return err
	}
	if err := tmux.SetPaneTitle(paneID, windowName); err != nil {
		return err
	}

	input := state.UpdateInput{
		RuntimeKey: runtimeKey,
		Source:     "codex",
		PaneID:     paneID,
		Status:     state.StatusStarting,
		Repo:       currentPath,
		CWD:        currentPath,
		Title:      windowName,
	}
	st, err := loadOrCreateState(input)
	if err != nil {
		return err
	}
	if err := enrichStateFromTMUX(&st); err != nil {
		return err
	}
	applyAgentOwnership(&st, ownerSessionID, ownerSessionName, ownerWindowID, ownerWindowName)
	if err := state.Save(st); err != nil {
		return err
	}
	if err := syncTMUX(st, paneID); err != nil {
		return err
	}
	return tmux.FocusPane(st.TmuxSession, st.TmuxWindow, paneID)
}

func runPrepare(args []string) error {
	fs := flag.NewFlagSet("prepare", flag.ContinueOnError)
	input, paneID, metadata, err := parseCommonFlags(fs, args, true)
	if err != nil {
		return err
	}
	st, err := loadOrCreateState(input)
	if err != nil {
		return err
	}
	if err := enrichStateFromTMUX(&st); err != nil {
		return err
	}
	if len(metadata) > 0 {
		st.Metadata = metadata
	}
	if err := state.Save(st); err != nil {
		return err
	}
	return syncTMUX(st, paneID)
}

func runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	input, paneID, metadata, err := parseCommonFlags(fs, args, true)
	if err != nil {
		return err
	}
	st, err := loadOrCreateState(input)
	if err != nil {
		return err
	}
	if err := enrichStateFromTMUX(&st); err != nil {
		return err
	}
	if len(metadata) > 0 {
		st.Metadata = metadata
	}
	if err := state.Save(st); err != nil {
		return err
	}
	return syncTMUX(st, paneID)
}

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	input, paneID, metadata, err := parseCommonFlags(fs, args, false)
	if err != nil {
		return err
	}
	st, err := state.Load(input.RuntimeKey)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	input.Metadata = metadata
	st.ApplyUpdate(input, false)
	if err := enrichStateFromTMUX(&st); err != nil {
		return err
	}
	if err := state.Save(st); err != nil {
		return err
	}
	return syncTMUX(st, paneID)
}

func runFinish(args []string) error {
	fs := flag.NewFlagSet("finish", flag.ContinueOnError)
	input, paneID, metadata, err := parseCommonFlags(fs, args, false)
	if err != nil {
		return err
	}
	st, err := state.Load(input.RuntimeKey)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	input.Metadata = metadata
	st.ApplyUpdate(input, true)
	if err := enrichStateFromTMUX(&st); err != nil {
		return err
	}
	if err := state.Save(st); err != nil {
		return err
	}
	return syncTMUX(st, paneID)
}

func runCleanup(args []string) error {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	runtimeKey := fs.String("runtime-key", "", "runtime key")
	paneID := fs.String("pane-id", os.Getenv("TMUX_PANE"), "tmux pane id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *runtimeKey == "" {
		return errors.New("runtime-key is required")
	}
	_ = tmux.SetPaneOption(*paneID, "@agent_role", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_source", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_runtime_key", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_status", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_updated_at", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_title", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_owner_session_id", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_owner_session_name", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_owner_window_id", "")
	_ = tmux.SetPaneOption(*paneID, "@agent_owner_window_name", "")
	return state.Delete(*runtimeKey)
}

func runStatusSegment() error {
	agentSessionName, err := tmux.ShowOption("@agent-sidebar-agent-session-name")
	if err != nil || agentSessionName == "" {
		agentSessionName = "__agent__"
	}

	panes, err := tmux.ListPanes()
	if err != nil {
		return err
	}

	windowStatuses := map[string]string{}
	hasAttention := false
	hasWaiting := false
	for _, pane := range panes {
		if pane.SessionName != agentSessionName {
			continue
		}
		status := pane.AgentStatus
		if status == "" {
			status = state.StatusIdle
		}
		if pane.RuntimeKey != "" {
			if st, err := state.Load(pane.RuntimeKey); err == nil && st.Status != "" {
				status = st.Status
			}
		}
		if current, ok := windowStatuses[pane.WindowID]; !ok || statusPriority(status) > statusPriority(current) {
			windowStatuses[pane.WindowID] = status
		}
	}

	for _, status := range windowStatuses {
		switch status {
		case state.StatusError:
			hasAttention = true
		case state.StatusWaitingInput:
			hasWaiting = true
		}
	}

	count := len(windowStatuses)
	style := "#[fg=8]"
	suffix := ""
	switch {
	case hasAttention:
		style = "#[fg=9]"
		suffix = "!"
	case hasWaiting:
		style = "#[fg=11]"
		suffix = "?"
	case count > 0:
		style = "#[fg=10]"
	}

	fmt.Printf("%sA:%d%s#[default]", style, count, suffix)
	return nil
}

func statusPriority(status string) int {
	switch status {
	case state.StatusError:
		return 5
	case state.StatusWaitingInput:
		return 4
	case state.StatusRunning:
		return 3
	case state.StatusStarting:
		return 2
	case state.StatusSuccess, state.StatusIdle:
		return 1
	default:
		return 0
	}
}

func parseCommonFlags(fs *flag.FlagSet, args []string, requireStartFields bool) (state.UpdateInput, string, map[string]string, error) {
	var input state.UpdateInput
	source := fs.String("source", "", "agent source")
	paneID := fs.String("pane-id", os.Getenv("TMUX_PANE"), "tmux pane id")
	runtimeKey := fs.String("runtime-key", "", "runtime key")
	status := fs.String("status", "", "agent status")
	repo := fs.String("repo", "", "repo path")
	cwd := fs.String("cwd", "", "working directory")
	title := fs.String("title", "", "title")
	taskSummary := fs.String("task-summary", "", "task summary")
	sessionID := fs.String("session-id", "", "session id")
	parentID := fs.String("parent-id", "", "parent agent id")
	model := fs.String("model", "", "model")
	lastError := fs.String("last-error", "", "last error")
	metadataValues := multiFlag{}
	fs.Var(&metadataValues, "metadata", "metadata key=value")
	if err := fs.Parse(args); err != nil {
		return input, "", nil, err
	}
	if *runtimeKey == "" {
		return input, "", nil, errors.New("runtime-key is required")
	}
	if requireStartFields {
		if *source == "" {
			return input, "", nil, errors.New("source is required")
		}
		if *paneID == "" {
			return input, "", nil, errors.New("pane-id is required")
		}
	}
	input = state.UpdateInput{
		RuntimeKey:  *runtimeKey,
		Source:      *source,
		PaneID:      *paneID,
		Status:      *status,
		Repo:        *repo,
		CWD:         *cwd,
		Title:       *title,
		TaskSummary: *taskSummary,
		SessionID:   *sessionID,
		ParentID:    *parentID,
		Model:       *model,
		LastError:   *lastError,
		Metadata:    parseMetadata(metadataValues),
	}
	if input.Status == "" && requireStartFields {
		input.Status = state.StatusStarting
	}
	return input, *paneID, input.Metadata, nil
}

func loadOrCreateState(input state.UpdateInput) (state.RuntimeState, error) {
	st, err := state.Load(input.RuntimeKey)
	if err == nil {
		st.ApplyUpdate(input, false)
		return st, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return state.RuntimeState{}, fmt.Errorf("load state: %w", err)
	}
	return state.NewRuntimeState(input), nil
}

func enrichStateFromTMUX(st *state.RuntimeState) error {
	panes, err := tmux.ListPanes()
	if err != nil {
		return nil
	}
	for _, pane := range panes {
		if pane.PaneID != st.PaneID {
			continue
		}
		st.TmuxSession = pane.SessionID
		if st.Metadata == nil {
			st.Metadata = map[string]string{}
		}
		st.Metadata["tmux_session_name"] = pane.SessionName
		st.TmuxWindow = pane.WindowID
		st.TmuxWindowName = pane.WindowName
		if st.CWD == "" {
			st.CWD = pane.CurrentPath
		}
		break
	}
	return nil
}

func syncTMUX(st state.RuntimeState, paneID string) error {
	targetPane := paneID
	if targetPane == "" {
		targetPane = st.PaneID
	}
	if targetPane == "" {
		return nil
	}
	updates := [][2]string{
		{"@agent_role", "agent"},
		{"@agent_source", st.Source},
		{"@agent_runtime_key", st.RuntimeKey},
		{"@agent_status", st.Status},
		{"@agent_updated_at", st.UpdatedAt},
		{"@agent_title", st.Title},
		{"@agent_owner_session_id", st.Metadata["owner_session_id"]},
		{"@agent_owner_session_name", st.Metadata["owner_session_name"]},
		{"@agent_owner_window_id", st.Metadata["owner_window_id"]},
		{"@agent_owner_window_name", st.Metadata["owner_window_name"]},
	}
	for _, kv := range updates {
		if err := tmux.SetPaneOption(targetPane, kv[0], kv[1]); err != nil {
			return err
		}
	}
	agentSessionName, err := tmux.ShowOption("@agent-sidebar-agent-session-name")
	if err != nil || agentSessionName == "" {
		agentSessionName = "__agent__"
	}
	if st.Metadata["tmux_session_name"] == agentSessionName && st.TmuxSession != "" {
		_ = tmux.SetSessionOption(st.TmuxSession, "@session_kind", "agent")
		_ = tmux.SetSessionOption(st.TmuxSession, "detach-on-destroy", "off")
		_ = tmux.SetSessionOption(st.TmuxSession, "@agent_owner_session_id", st.Metadata["owner_session_id"])
		_ = tmux.SetSessionOption(st.TmuxSession, "@agent_owner_session_name", st.Metadata["owner_session_name"])
		_ = tmux.SetSessionOption(st.TmuxSession, "@agent_owner_window_id", st.Metadata["owner_window_id"])
		_ = tmux.SetSessionOption(st.TmuxSession, "@agent_owner_window_name", st.Metadata["owner_window_name"])
	}
	if st.Metadata["tmux_session_name"] == agentSessionName && st.TmuxWindow != "" {
		_ = tmux.SetWindowOption(st.TmuxWindow, "@window_kind", "agent")
	}
	return nil
}

func applyAgentOwnership(st *state.RuntimeState, ownerSessionID, ownerSessionName, ownerWindowID, ownerWindowName string) {
	if st.Metadata == nil {
		st.Metadata = map[string]string{}
	}
	if ownerSessionID != "" {
		st.Metadata["owner_session_id"] = ownerSessionID
	}
	if ownerSessionName != "" {
		st.Metadata["owner_session_name"] = ownerSessionName
	}
	if ownerWindowID != "" {
		st.Metadata["owner_window_id"] = ownerWindowID
	}
	if ownerWindowName != "" {
		st.Metadata["owner_window_name"] = ownerWindowName
	}
}

func findPaneByID(panes []tmux.Pane, paneID string) *tmux.Pane {
	for i := range panes {
		if panes[i].PaneID == paneID {
			return &panes[i]
		}
	}
	return nil
}

func resolveNormalWindowTarget(currentPane *tmux.Pane) (paneID, sessionTarget, windowID, windowName string) {
	if currentPane == nil {
		return "", "", "", ""
	}
	paneID = ""
	sessionTarget = currentPane.OwnerSessionID
	windowID = currentPane.OwnerWindowID
	if sessionTarget == "" {
		sessionTarget, _ = tmux.ShowPaneOption(currentPane.PaneID, "@agent_owner_session_name")
	}
	windowName, _ = tmux.ShowPaneOption(currentPane.PaneID, "@agent_owner_window_name")
	if windowName == "" {
		windowName = "main"
	}

	panes, err := tmux.ListPanes()
	if err != nil {
		return "", sessionTarget, windowID, windowName
	}
	for _, pane := range panes {
		if pane.SessionID == sessionTarget && pane.WindowID == windowID {
			return pane.PaneID, sessionTarget, windowID, windowName
		}
	}
	return "", sessionTarget, windowID, windowName
}

func parseMetadata(values []string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, item := range values {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func selfExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		exe = filepath.Clean(exe)
	}
	return exe, nil
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func usage(name string) {
	fmt.Fprintf(os.Stderr, "usage: %s <sidebar|status-segment|toggle|open|close|switch-window|prepare|start|update|finish|cleanup> [flags]\n", name)
}

func Exit(args []string) {
	if err := Run(args); err != nil {
		log.SetOutput(os.Stderr)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
