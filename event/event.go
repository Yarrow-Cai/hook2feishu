package event

import (
	"encoding/json"
	"io"
	"os"
)

// Tool identifies which programming tool sent the hook event.
type Tool string

const (
	ToolClaudeCode Tool = "claude"
	ToolCodex      Tool = "codex"
	ToolUnknown    Tool = "unknown"
)

// Event represents a normalized hook event from any supported tool.
type Event struct {
	Tool            Tool   // detected tool
	HookEventName   string // original event name
	CWD             string
	SessionID       string // Claude Code: session_id; Codex: task_id
	TranscriptPath  string // Claude Code only
	LastAssistantMessage string
	Message         string
	Title           string
	NotificationType string
}

// ReadStdin reads and auto-detects a hook event from stdin.
// Returns nil,nil when there's no input (tty or empty pipe).
func ReadStdin() (*Event, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	// Try Claude Code format
	if evt, ok := parseClaude(data); ok {
		return evt, nil
	}

	// Try Codex format
	if evt, ok := parseCodex(data); ok {
		return evt, nil
	}

	// Best-effort fallback
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &Event{Tool: ToolUnknown}, nil
}

// ── Claude Code parser ───────────────────────────────────────────

// claudeRaw mirrors the Claude Code hook JSON from stdin.
type claudeRaw struct {
	HookEventName        string `json:"hook_event_name"`
	CWD                  string `json:"cwd"`
	SessionID            string `json:"session_id"`
	TranscriptPath       string `json:"transcript_path"`
	LastAssistantMessage string `json:"last_assistant_message"`
	Message              string `json:"message"`
	Title                string `json:"title"`
	NotificationType     string `json:"notification_type"`
}

func parseClaude(data []byte) (*Event, bool) {
	var raw claudeRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false
	}
	// Detection key: Claude Code always has hook_event_name
	if raw.HookEventName == "" {
		return nil, false
	}
	return &Event{
		Tool:                ToolClaudeCode,
		HookEventName:       raw.HookEventName,
		CWD:                 raw.CWD,
		SessionID:           raw.SessionID,
		TranscriptPath:      raw.TranscriptPath,
		LastAssistantMessage: raw.LastAssistantMessage,
		Message:             raw.Message,
		Title:               raw.Title,
		NotificationType:    raw.NotificationType,
	}, true
}

// ── Codex parser ─────────────────────────────────────────────────

// codexRaw mirrors the Codex CLI hook JSON from stdin.
// Fields are based on Codex hook documentation.
type codexRaw struct {
	Event     string `json:"event"`      // "stop", "notification", "request"
	TaskID    string `json:"task_id"`    // maps to SessionID
	CWD       string `json:"cwd"`
	Message   string `json:"message"`
	Title     string `json:"title"`
	Type      string `json:"type"`       // notification type
	LastMessage string `json:"last_message"`
}

func parseCodex(data []byte) (*Event, bool) {
	var raw codexRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false
	}
	if raw.Event == "" {
		return nil, false
	}
	// Map Codex event names to Claude Code equivalents
	eventName := mapCodexEvent(raw.Event)
	return &Event{
		Tool:                ToolCodex,
		HookEventName:       eventName,
		CWD:                 raw.CWD,
		SessionID:           raw.TaskID,
		LastAssistantMessage: raw.LastMessage,
		Message:             raw.Message,
		Title:               raw.Title,
		NotificationType:    raw.Type,
	}, true
}

func mapCodexEvent(e string) string {
	switch e {
	case "stop":
		return "Stop"
	case "notification", "request":
		return "Notification"
	default:
		return e
	}
}
