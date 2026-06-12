package notifier

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LarkCLI wraps lark-cli for sending messages.
type LarkCLI struct {
	Path    string // lark-cli path; empty = auto-detect
	Profile string // optional profile name
}

// SendCard delivers an interactive card to the user.
func (l *LarkCLI) SendCard(card map[string]interface{}, openID string) error {
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}

	args := []string{
		"im", "+messages-send",
		"--as", "bot",
		"--user-id", openID,
		"--msg-type", "interactive",
		"--content", string(cardJSON),
	}
	return l.run(args)
}

// SendText sends a plain text message (fallback / debug).
func (l *LarkCLI) SendText(text string, openID string) error {
	args := []string{
		"im", "+messages-send",
		"--as", "bot",
		"--user-id", openID,
		"--msg-type", "text",
		"--text", text,
	}
	return l.run(args)
}

// SendMarkdown sends a markdown-formatted message.
func (l *LarkCLI) SendMarkdown(md string, openID string) error {
	args := []string{
		"im", "+messages-send",
		"--as", "bot",
		"--user-id", openID,
		"--markdown", md,
	}
	return l.run(args)
}

func (l *LarkCLI) run(subcmdArgs []string) error {
	exe, allArgs := l.resolve(subcmdArgs)

	cmd := exec.Command(exe, allArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lark-cli: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// resolve determines the actual executable and full argument list.
func (l *LarkCLI) resolve(subcmdArgs []string) (string, []string) {
	var args []string
	if l.Profile != "" {
		args = append(args, "--profile", l.Profile)
	}
	args = append(args, subcmdArgs...)

	path := l.Path
	if path == "" {
		path = "lark-cli"
	}

	// On Windows, prefer node + entry.js over shell/cmd wrappers
	if runtime.GOOS == "windows" {
		base := filepath.Base(path)
		base = strings.TrimSuffix(base, filepath.Ext(base)) // strip .cmd/.bat/.sh
		if base == "lark-cli" || base == "lark_cli" {
			if node, entry, ok := findNodeEntry(); ok {
				// Build: node entry.js --profile ... im +messages-send ...
				fullArgs := append([]string{entry}, args...)
				return node, fullArgs
			}
		}
	}

	return path, args
}

// ── Windows node + lark-cli entry discovery ──────────────────────

func findNodeEntry() (string, string, bool) {
	node := findNode()
	if node == "" {
		return "", "", false
	}

	entry := findLarkCLIEntry()
	if entry == "" {
		return "", "", false
	}

	return node, entry, true
}

func findNode() string {
	candidates := []string{
		`D:\Program Files\nodejs\node.exe`,
		`C:\Program Files\nodejs\node.exe`,
	}

	// Try npm root's bundled node
	if root := npmRoot(); root != "" {
		candidates = append(candidates, filepath.Join(root, "node.exe"))
	}

	// Check explicit paths
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Fallback: PATH lookup
	if _, err := exec.LookPath("node"); err == nil {
		return "node"
	}

	return ""
}

func findLarkCLIEntry() string {
	candidates := []string{}

	if root := npmRoot(); root != "" {
		candidates = append(candidates,
			filepath.Join(root, "node_modules", "@larksuite", "cli", "scripts", "run.js"))
	}

	// Additional npm global roots
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "AppData", "Roaming", "npm", "node_modules", "@larksuite", "cli", "scripts", "run.js"),
			filepath.Join(home, "AppData", "Local", "npm", "node_modules", "@larksuite", "cli", "scripts", "run.js"),
		)
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}

func npmRoot() string {
	if home, err := os.UserHomeDir(); err == nil {
		root := filepath.Join(home, "AppData", "Roaming", "npm")
		if _, err := os.Stat(root); err == nil {
			return root
		}
	}
	return ""
}
