package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/liubog2008/tmux-agent/internal/state"
	"github.com/liubog2008/tmux-agent/internal/tmux"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "start":
		err = runStart(os.Args[2:])
	case "update":
		err = runUpdate(os.Args[2:])
	case "finish":
		err = runFinish(os.Args[2:])
	case "cleanup":
		err = runCleanup(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	input, paneID, metadata, err := parseCommonFlags(fs, args, true)
	if err != nil {
		return err
	}
	st := state.NewRuntimeState(input)
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

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <start|update|finish|cleanup> [flags]\n", os.Args[0])
}
