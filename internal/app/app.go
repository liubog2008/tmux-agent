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
	case "toggle":
		err = runToggle()
	case "open":
		err = runOpen()
	case "close":
		err = runClose(args[2:])
	case "new-window":
		err = runNewWindow()
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

func runNewWindow() error {
	exe, err := selfExecutable()
	if err != nil {
		return err
	}

	windowName, err := tmux.ShowOption("@agent-sidebar-agent-window-name")
	if err != nil || windowName == "" {
		windowName = "agent"
	}

	currentPath, err := tmux.Format("#{pane_current_path}")
	if err != nil || currentPath == "" {
		currentPath, _ = os.Getwd()
	}

	defaultShell, err := tmux.ShowOption("default-shell")
	if err != nil || defaultShell == "" {
		defaultShell = os.Getenv("SHELL")
	}
	if defaultShell == "" {
		defaultShell = "/bin/sh"
	}

	runtimeKey := fmt.Sprintf("agent-%d-%d", time.Now().Unix(), os.Getpid())
	paneID, err := tmux.NewWindow(
		currentPath,
		windowName,
		"env",
		"TMUX_AGENT_RUNTIME_KEY="+runtimeKey,
		"TMUX_AGENT_BIN="+exe,
		defaultShell,
		"-l",
	)
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
	if err := state.Save(st); err != nil {
		return err
	}
	return syncTMUX(st, paneID)
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
	return state.Delete(*runtimeKey)
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
	}
	for _, kv := range updates {
		if err := tmux.SetPaneOption(targetPane, kv[0], kv[1]); err != nil {
			return err
		}
	}
	if st.TmuxSession != "" {
		_ = tmux.SetSessionOption(st.TmuxSession, "@session_kind", "agent")
	}
	return nil
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
	fmt.Fprintf(os.Stderr, "usage: %s <sidebar|toggle|open|close|new-window|prepare|start|update|finish|cleanup> [flags]\n", name)
}

func Exit(args []string) {
	if err := Run(args); err != nil {
		log.SetOutput(os.Stderr)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
