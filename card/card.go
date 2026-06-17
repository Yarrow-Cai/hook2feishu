package card

import (
	"fmt"
	"path/filepath"
)

// Card is a Feishu interactive card.
type Card struct {
	Config   CardConfig   `json:"config"`
	Header   CardHeader   `json:"header"`
	Elements []CardElement `json:"elements"`
}

type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type CardHeader struct {
	Title    CardText `json:"title"`
	Template string   `json:"template"`
}

type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type CardElement map[string]interface{}

// AgentInfo matches transcript.AgentInfo.
type AgentInfo struct {
	Description string
	Type        string
	Name        string
}

// ── Tool labels ──────────────────────────────────────────────────

func toolLabel(tool string) string {
	switch tool {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude Code"
	default:
		return "AI 助手"
	}
}

func toolEmoji(tool string) string {
	switch tool {
	case "codex":
		return "🤖"
	case "claude":
		return "🧠"
	default:
		return "✨"
	}
}

// ── Stop card ────────────────────────────────────────────────────

type StopCardParams struct {
	Tool           string
	Project        string
	Host           string
	Duration       string
	TotalTokens    int
	TurnTokens     int
	TotalTools     int
	TurnTools      int
	Turns          int
	Branch         string
	Dirty          bool
	LastCommit     string
	NewAgents      []AgentInfo
	NewAgentCount  int
	TotalAgents    int
	PrevAgentCount int
	LastMessage    string
	CWD            string
	SessionID      string
	Now            string
	IsSubAgent     bool
}

func BuildStopCard(params StopCardParams) *Card {
	elements := []CardElement{}

	// Row 1: Project + Device
	elements = append(elements, columns(
		column(1, fmt.Sprintf("📁 **项目**\n%s", params.Project)),
		column(1, fmt.Sprintf("💻 **设备**\n%s", params.Host)),
	))
	elements = append(elements, hr())

	// Stats row
	statsCols := []CardElement{}
	if params.Duration != "" {
		statsCols = append(statsCols, column(1, fmt.Sprintf("⏱ **耗时**\n%s", params.Duration)))
	}
	if params.TotalTokens > 0 {
		statsCols = append(statsCols, column(1, fmt.Sprintf("📊 **Tokens**\n%s / %s",
			fmtTokens(params.TurnTokens), fmtTokens(params.TotalTokens))))
	}
	if params.TotalTools > 0 {
		statsCols = append(statsCols, column(1, fmt.Sprintf("🔧 **工具**\n%d / %d",
			params.TurnTools, params.TotalTools)))
	}
	if params.Turns > 0 {
		statsCols = append(statsCols, column(1, fmt.Sprintf("💬 **对话**\n%d 轮", params.Turns)))
	}
	if params.Branch != "" {
		dirtyMark := ""
		if params.Dirty {
			dirtyMark = " ●"
		}
		statsCols = append(statsCols, column(1, fmt.Sprintf("🌿 **分支**\n`%s`%s", params.Branch, dirtyMark)))
	}
	if len(statsCols) > 0 {
		elements = append(elements, columns(statsCols...))
	}

	// Sub-agents
	if params.NewAgentCount > 0 && len(params.NewAgents) > 0 {
		lines := []string{}
		for _, a := range params.NewAgents {
			label := a.Name
			if label == "" {
				label = a.Type
			}
			if label == "" {
				label = "agent"
			}
			if a.Description != "" {
				lines = append(lines, fmt.Sprintf("• **%s**  %s", label, a.Description))
			} else {
				lines = append(lines, fmt.Sprintf("• **%s**", label))
			}
		}
		header := fmt.Sprintf("🤖 **子 Agent** (%d 个", params.NewAgentCount)
		if params.PrevAgentCount > 0 {
			header += fmt.Sprintf(" / 本会话共 %d", params.TotalAgents)
		}
		header += ")"
		lines = append([]string{header}, lines...)
		elements = append(elements, markdown(joinLines(lines)))
	}

	// Git last commit
	if params.LastCommit != "" {
		elements = append(elements, markdown(fmt.Sprintf("📝 **最近提交**  `%s`", params.LastCommit)))
	}
	elements = append(elements, hr())

	// Last message
	if params.LastMessage != "" {
		cleaned := CleanMarkdown(params.LastMessage)
		snippet := truncate(cleaned, 10000)
		elements = append(elements, markdown(fmt.Sprintf("💬 **回复**\n%s", snippet)))
		elements = append(elements, hr())
	}

	// Footer
	footerParts := []string{}
	if params.CWD != "" {
		footerParts = append(footerParts, fmt.Sprintf("📂 %s", params.CWD))
	}
	if params.SessionID != "" {
		s := params.SessionID
		if len(s) > 12 {
			s = s[:12]
		}
		footerParts = append(footerParts, fmt.Sprintf("🔑 %s", s))
	}
	footerParts = append(footerParts, fmt.Sprintf("🕐 %s", params.Now))
	elements = append(elements, note(joinParts(footerParts, "  |  ")))

	// Header — tool-agnostic
	headerTitle := fmt.Sprintf("%s %s 任务完成", toolEmoji(params.Tool), toolLabel(params.Tool))
	headerColor := "turquoise"
	if params.IsSubAgent {
		worktreeName := ""
		if params.CWD != "" {
			worktreeName = filepath.Base(params.CWD)
		}
		if worktreeName != "" && worktreeName != params.Project {
			headerTitle = fmt.Sprintf("🔧 子 Agent 完成 — %s/%s", params.Project, worktreeName)
		} else {
			headerTitle = fmt.Sprintf("🔧 子 Agent 完成 — %s", params.Project)
		}
		headerColor = "blue"
	}

	return &Card{
		Config: CardConfig{WideScreenMode: true},
		Header: CardHeader{
			Title:    CardText{Tag: "plain_text", Content: headerTitle},
			Template: headerColor,
		},
		Elements: elements,
	}
}

// ── Notification card ────────────────────────────────────────────

type NotifCardParams struct {
	Tool     string
	Project  string
	Host     string
	Message  string
	Title    string
	NotifType string
	Branch   string
	CWD      string
	Now      string
}

func BuildNotificationCard(params NotifCardParams) *Card {
	headerMap := map[string][2]string{
		"permission_prompt":  {"⚠️ 需要你的确认", "orange"},
		"idle_prompt":        {"⏳ 等待输入", "yellow"},
		"auth_success":       {"✅ 认证成功", "green"},
		"elicitation_dialog": {"📝 需要信息", "blue"},
	}
	headerTitle := fmt.Sprintf("🔔 %s 通知", toolLabel(params.Tool))
	headerColor := "blue"
	if pair, ok := headerMap[params.NotifType]; ok {
		headerTitle = pair[0]
		headerColor = pair[1]
	}

	elements := []CardElement{}

	// Row 1: Project + Device
	elements = append(elements, columns(
		column(1, fmt.Sprintf("📁 **项目**\n%s", params.Project)),
		column(1, fmt.Sprintf("💻 **设备**\n%s", params.Host)),
	))
	elements = append(elements, hr())

	// Title + Message
	if params.Title != "" {
		elements = append(elements, markdown(fmt.Sprintf("**%s**", params.Title)))
	}
	if params.Message != "" {
		elements = append(elements, markdown(fmt.Sprintf("💬 %s", truncate(CleanMarkdown(params.Message), 10000))))
	}

	// Branch
	if params.Branch != "" {
		elements = append(elements, markdown(fmt.Sprintf("🌿 **分支**  `%s`", params.Branch)))
	}
	elements = append(elements, hr())

	// Footer
	footerParts := []string{}
	if params.CWD != "" {
		footerParts = append(footerParts, fmt.Sprintf("📂 %s", params.CWD))
	}
	footerParts = append(footerParts, fmt.Sprintf("🕐 %s", params.Now))
	elements = append(elements, note(joinParts(footerParts, "  |  ")))

	return &Card{
		Config: CardConfig{WideScreenMode: true},
		Header: CardHeader{
			Title:    CardText{Tag: "plain_text", Content: headerTitle},
			Template: headerColor,
		},
		Elements: elements,
	}
}

// ── Element helpers ──────────────────────────────────────────────

func column(weight int, content string) CardElement {
	return CardElement{
		"tag":            "column",
		"width":          "weighted",
		"weight":         weight,
		"vertical_align": "top",
		"elements":       []CardElement{{"tag": "markdown", "content": content}},
	}
}

func columns(cols ...CardElement) CardElement {
	return CardElement{
		"tag":              "column_set",
		"flex_mode":        "none",
		"background_style": "default",
		"columns":          cols,
	}
}

func hr() CardElement {
	return CardElement{"tag": "hr"}
}

func markdown(content string) CardElement {
	return CardElement{"tag": "markdown", "content": content}
}

func note(content string) CardElement {
	return CardElement{
		"tag": "note",
		"elements": []CardElement{
			{"tag": "plain_text", "content": content},
		},
	}
}

// ── Formatting helpers ───────────────────────────────────────────

func fmtTokens(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1000000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
}

func truncate(s string, maxLen int) string {
	s = trimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		if result != "" {
			result += "\n"
		}
		result += line
	}
	return result
}

func joinParts(parts []string, sep string) string {
	result := ""
	for _, p := range parts {
		if result != "" {
			result += sep
		}
		result += p
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
