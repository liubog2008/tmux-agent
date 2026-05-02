package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	Version = 1

	StatusStarting     = "starting"
	StatusRunning      = "running"
	StatusWaitingInput = "waiting_input"
	StatusIdle         = "idle"
	StatusSuccess      = "success"
	StatusError        = "error"
	StatusStale        = "stale"
)

type RuntimeState struct {
	Version        int               `json:"version"`
	RuntimeKey     string            `json:"runtime_key"`
	Source         string            `json:"source"`
	PaneID         string            `json:"pane_id"`
	TmuxSession    string            `json:"tmux_session,omitempty"`
	TmuxWindow     string            `json:"tmux_window,omitempty"`
	TmuxWindowName string            `json:"tmux_window_name,omitempty"`
	Repo           string            `json:"repo,omitempty"`
	CWD            string            `json:"cwd,omitempty"`
	Status         string            `json:"status"`
	Title          string            `json:"title,omitempty"`
	TaskSummary    string            `json:"task_summary,omitempty"`
	SessionID      string            `json:"session_id,omitempty"`
	ParentAgentID  string            `json:"parent_agent_id,omitempty"`
	Model          string            `json:"model,omitempty"`
	StartedAt      string            `json:"started_at"`
	UpdatedAt      string            `json:"updated_at"`
	EndedAt        string            `json:"ended_at,omitempty"`
	LastError      string            `json:"last_error,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type UpdateInput struct {
	RuntimeKey  string
	Source      string
	PaneID      string
	Status      string
	Repo        string
	CWD         string
	Title       string
	TaskSummary string
	SessionID   string
	ParentID    string
	Model       string
	LastError   string
	Metadata    map[string]string
}

func NewRuntimeState(input UpdateInput) RuntimeState {
	now := time.Now().Format(time.RFC3339)
	return RuntimeState{
		Version:       Version,
		RuntimeKey:    input.RuntimeKey,
		Source:        input.Source,
		PaneID:        input.PaneID,
		Repo:          input.Repo,
		CWD:           input.CWD,
		Status:        input.Status,
		Title:         input.Title,
		TaskSummary:   input.TaskSummary,
		SessionID:     input.SessionID,
		ParentAgentID: input.ParentID,
		Model:         input.Model,
		LastError:     input.LastError,
		Metadata:      cloneMap(input.Metadata),
		StartedAt:     now,
		UpdatedAt:     now,
	}
}

func (s *RuntimeState) ApplyUpdate(input UpdateInput, terminal bool) {
	now := time.Now().Format(time.RFC3339)
	if input.Source != "" {
		s.Source = input.Source
	}
	if input.PaneID != "" {
		s.PaneID = input.PaneID
	}
	if input.Status != "" {
		s.Status = input.Status
	}
	if input.Repo != "" {
		s.Repo = input.Repo
	}
	if input.CWD != "" {
		s.CWD = input.CWD
	}
	if input.Title != "" {
		s.Title = input.Title
	}
	if input.TaskSummary != "" {
		s.TaskSummary = input.TaskSummary
	}
	if input.SessionID != "" {
		s.SessionID = input.SessionID
	}
	if input.ParentID != "" {
		s.ParentAgentID = input.ParentID
	}
	if input.Model != "" {
		s.Model = input.Model
	}
	if input.LastError != "" {
		s.LastError = input.LastError
	}
	if len(input.Metadata) > 0 {
		if s.Metadata == nil {
			s.Metadata = map[string]string{}
		}
		for k, v := range input.Metadata {
			s.Metadata[k] = v
		}
	}
	s.UpdatedAt = now
	if terminal {
		s.EndedAt = now
	}
}

func RuntimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "tmux-agent-sidebar")
	}
	return filepath.Join(os.TempDir(), "tmux-agent-sidebar")
}

func RuntimePath(runtimeKey string) string {
	return filepath.Join(RuntimeDir(), runtimeKey+".json")
}

func EnsureRuntimeDir() error {
	return os.MkdirAll(RuntimeDir(), 0o755)
}

func Load(runtimeKey string) (RuntimeState, error) {
	var st RuntimeState
	path := RuntimePath(runtimeKey)
	data, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	err = json.Unmarshal(data, &st)
	return st, err
}

func Save(st RuntimeState) error {
	if st.RuntimeKey == "" {
		return errors.New("runtime_key is required")
	}
	if err := EnsureRuntimeDir(); err != nil {
		return err
	}
	path := RuntimePath(st.RuntimeKey)
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime state: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func Delete(runtimeKey string) error {
	err := os.Remove(RuntimePath(runtimeKey))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
