package notifier

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// LarkCLI wraps lark-cli for sending messages.
type LarkCLI struct {
	Path    string // lark-cli binary path, empty = find in PATH
	Profile string // optional profile name
}

// SendCard delivers an interactive card to the user.
func (l *LarkCLI) SendCard(card map[string]interface{}, openID string) error {
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}

	args := l.baseArgs("+messages-send")
	args = append(args,
		"--as", "user",
		"--user-id", openID,
		"--msg-type", "interactive",
		"--content", string(cardJSON),
	)
	return l.run(args)
}

// SendText sends a plain text message (fallback / debug).
func (l *LarkCLI) SendText(text string, openID string) error {
	args := l.baseArgs("+messages-send")
	args = append(args,
		"--as", "user",
		"--user-id", openID,
		"--msg-type", "text",
		"--text", text,
	)
	return l.run(args)
}

// SendMarkdown sends a markdown-formatted message.
func (l *LarkCLI) SendMarkdown(md string, openID string) error {
	args := l.baseArgs("+messages-send")
	args = append(args,
		"--as", "user",
		"--user-id", openID,
		"--markdown", md,
	)
	return l.run(args)
}

func (l *LarkCLI) baseArgs(subcmd string) []string {
	cmd := l.Path
	if cmd == "" {
		cmd = "lark-cli"
	}
	args := []string{cmd}
	if l.Profile != "" {
		args = append(args, "--profile", l.Profile)
	}
	args = append(args, "im", subcmd)
	return args
}

func (l *LarkCLI) run(args []string) error {
	exe := args[0]
	rest := args[1:]

	cmd := exec.Command(exe, rest...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lark-cli: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
