package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yarrow-Cai/hook2feishu/card"
	"github.com/Yarrow-Cai/hook2feishu/checkpoint"
	"github.com/Yarrow-Cai/hook2feishu/config"
	"github.com/Yarrow-Cai/hook2feishu/debug"
	"github.com/Yarrow-Cai/hook2feishu/event"
	gitcmd "github.com/Yarrow-Cai/hook2feishu/git"
	"github.com/Yarrow-Cai/hook2feishu/notifier"
	"github.com/Yarrow-Cai/hook2feishu/sanitize"
	"github.com/Yarrow-Cai/hook2feishu/transcript"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hook2feishu: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	debug.Log("=== hook2feishu start ===")

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		debug.Log("config load failed: %v", err)
		return nil // silent
	}
	debug.Log("config loaded: open_id=%s profile=%s", cfg.OpenID, cfg.LarkCLIProfile)

	// 2. Read stdin event (auto-detect tool)
	evt, err := event.ReadStdin()
	if err != nil {
		debug.Log("stdin read error: %v", err)
		return nil
	}
	if evt == nil || evt.HookEventName == "" {
		debug.Log("no event data, exiting")
		return nil
	}
	debug.Log("event: tool=%s name=%s cwd=%s", evt.Tool, evt.HookEventName, evt.CWD)

	// 3. Event filter
	allowed := cfg.Events
	if len(allowed) == 0 {
		allowed = []string{"Stop", "Notification"}
	}
	if !contains(allowed, evt.HookEventName) {
		debug.Log("event '%s' not in allowed list %v, skipping", evt.HookEventName, allowed)
		return nil
	}

	// 4. Quiet hours check (Stop events only)
	if len(cfg.QuietHours) == 2 && evt.HookEventName == "Stop" {
		offset := cfg.TZOffset
		if offset == 0 {
			offset = 8
		}
		localHour := (time.Now().UTC().Hour() + offset) % 24
		start, end := cfg.QuietHours[0], cfg.QuietHours[1]
		isQuiet := false
		if start > end {
			isQuiet = localHour >= start || localHour < end
		} else {
			isQuiet = start <= localHour && localHour < end
		}
		if isQuiet {
			debug.Log("quiet hours (%d-%d), local hour=%d, skipping", start, end, localHour)
			return nil
		}
	}

	// 5. Parse transcript (Claude Code only — Codex doesn't have transcript)
	var stats *transcript.Stats
	if evt.Tool == event.ToolClaudeCode && evt.TranscriptPath != "" {
		stats = transcript.Parse(evt.TranscriptPath)
		debug.Log("stats: tokens=%d tools=%d turns=%d agents=%d",
			stats.TotalOutputTokens, stats.TotalToolCalls, stats.TotalTurns, stats.TotalAgents)
	} else {
		stats = &transcript.Stats{}
	}

	// 6. Min duration check (Stop events)
	if cfg.MinDuration > 0 && evt.HookEventName == "Stop" && stats.LastUserTS != "" {
		if elapsed := durationSeconds(stats.LastUserTS); elapsed > 0 && elapsed < cfg.MinDuration {
			debug.Log("duration %ds < min %ds, skipping", elapsed, cfg.MinDuration)
			return nil
		}
	}

	// 7. Get git info
	git := gitcmd.GetInfo(evt.CWD)
	debug.Log("git: branch=%s commit=%s dirty=%v", git.Branch, git.LastCommit, git.Dirty)

	// 8. Build card
	sessionID := evt.SessionID
	prev := checkpoint.Load(sessionID)
	now := formatNow(cfg.TZOffset)
	toolStr := string(evt.Tool)

	var cardData *card.Card

	if evt.HookEventName == "Stop" {
		turnTokens := 0
		turnTools := 0
		prevAgentCount := 0
		newAgentCount := 0
		var newAgents []card.AgentInfo

		if stats != nil && evt.Tool == event.ToolClaudeCode {
			turnTokens = stats.TotalOutputTokens - prev.OutputTokens
			if turnTokens < 0 {
				turnTokens = stats.TotalOutputTokens
			}
			turnTools = stats.TotalToolCalls - prev.ToolCalls
			if turnTools < 0 {
				turnTools = stats.TotalToolCalls
			}
			prevAgentCount = prev.Agents
			newAgentCount = stats.TotalAgents - prevAgentCount
			if newAgentCount < 0 {
				newAgentCount = stats.TotalAgents
				prevAgentCount = 0
			}
			if newAgentCount > 0 && len(stats.Agents) >= newAgentCount {
				for _, a := range stats.Agents[len(stats.Agents)-newAgentCount:] {
					newAgents = append(newAgents, card.AgentInfo{
						Description: a.Description,
						Type:        a.Type,
						Name:        a.Name,
					})
				}
			}
		}

		duration := formatDuration(stats.LastUserTS)
		branch := git.Branch
		if branch == "" {
			branch = stats.GitBranch
		}

		isSub := isSubAgent(evt.CWD)
		project := projectName(evt.CWD)

		cardData = card.BuildStopCard(card.StopCardParams{
			Tool:           toolStr,
			Project:        project,
			Host:           hostname(),
			Duration:       duration,
			TotalTokens:    stats.TotalOutputTokens,
			TurnTokens:     turnTokens,
			TotalTools:     stats.TotalToolCalls,
			TurnTools:      turnTools,
			Turns:          stats.TotalTurns,
			Branch:         branch,
			Dirty:          git.Dirty,
			LastCommit:     git.LastCommit,
			NewAgents:      newAgents,
			NewAgentCount:  newAgentCount,
			TotalAgents:    stats.TotalAgents,
			PrevAgentCount: prevAgentCount,
			LastMessage:    evt.LastAssistantMessage,
			CWD:            evt.CWD,
			SessionID:      sessionID,
			Now:            now,
			IsSubAgent:     isSub,
		})
	} else {
		project := projectName(evt.CWD)
		branch := git.Branch
		if branch == "" {
			branch = stats.GitBranch
		}
		cardData = card.BuildNotificationCard(card.NotifCardParams{
			Tool:      toolStr,
			Project:   project,
			Host:      hostname(),
			Message:   evt.Message,
			Title:     evt.Title,
			NotifType: evt.NotificationType,
			Branch:    branch,
			CWD:       evt.CWD,
			Now:       now,
		})
	}

	// 9. Sanitize + send via lark-cli
	cardMap := cardToMap(cardData)
	sanitized := sanitize.Recursive(cardMap)

	lark := &notifier.LarkCLI{
		Path:    cfg.LarkCLIPath,
		Profile: cfg.LarkCLIProfile,
	}
	if err := lark.SendCard(sanitized.(map[string]interface{}), cfg.OpenID); err != nil {
		debug.Log("send card failed: %v", err)
		return nil
	}
	debug.Log("card sent successfully")

	// 10. Save checkpoint (Claude Code only)
	if evt.Tool == event.ToolClaudeCode && sessionID != "" {
		checkpoint.Save(sessionID, &checkpoint.Checkpoint{
			OutputTokens: stats.TotalOutputTokens,
			ToolCalls:    stats.TotalToolCalls,
			Turns:        stats.TotalTurns,
			Agents:       stats.TotalAgents,
			Time:         float64(time.Now().Unix()),
		})
	}

	debug.Log("=== hook2feishu end ===")
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	if idx := strings.Index(h, "."); idx >= 0 {
		h = h[:idx]
	}
	return h
}

func formatNow(offset int) string {
	t := time.Now().UTC().Add(time.Duration(offset) * time.Hour)
	return t.Format("2006-01-02 15:04:05")
}

func formatDuration(isoTS string) string {
	if isoTS == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02T15:04:05", isoTS[:19])
	if err != nil {
		return ""
	}
	elapsed := time.Since(t)
	if elapsed < 0 {
		return "0s"
	}
	secs := int(elapsed.Seconds())
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm %ds", secs/60, secs%60)
	default:
		return fmt.Sprintf("%dh %dm", secs/3600, (secs%3600)/60)
	}
}

func durationSeconds(isoTS string) int {
	if isoTS == "" {
		return 0
	}
	t, err := time.Parse("2006-01-02T15:04:05", isoTS[:19])
	if err != nil {
		return 0
	}
	return int(time.Since(t).Seconds())
}

func projectName(cwd string) string {
	if cwd == "" {
		return "unknown"
	}
	return filepath.Base(cwd)
}

func isSubAgent(cwd string) bool {
	norm := strings.ReplaceAll(cwd, "\\", "/")
	return strings.Contains(norm, "/worktrees/") || strings.Contains(norm, "/.worktree")
}

func cardToMap(c *card.Card) map[string]interface{} {
	m := make(map[string]interface{})
	m["config"] = map[string]interface{}{
		"wide_screen_mode": c.Config.WideScreenMode,
	}
	m["header"] = map[string]interface{}{
		"title": map[string]interface{}{
			"tag":     c.Header.Title.Tag,
			"content": c.Header.Title.Content,
		},
		"template": c.Header.Template,
	}
	elements := make([]interface{}, len(c.Elements))
	for i, el := range c.Elements {
		elements[i] = sanitizeCardElement(el)
	}
	m["elements"] = elements
	return m
}

func sanitizeCardElement(el card.CardElement) map[string]interface{} {
	out := make(map[string]interface{}, len(el))
	for k, v := range el {
		switch val := v.(type) {
		case []card.CardElement:
			arr := make([]interface{}, len(val))
			for i, item := range val {
				arr[i] = sanitizeCardElement(item)
			}
			out[k] = arr
		case card.CardElement:
			out[k] = sanitizeCardElement(val)
		default:
			out[k] = val
		}
	}
	return out
}
