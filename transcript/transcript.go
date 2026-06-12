package transcript

import (
	"bufio"
	"encoding/json"
	"os"
)

// Stats holds statistics parsed from a Claude Code session transcript.
type Stats struct {
	TotalOutputTokens int         `json:"total_output_tokens"`
	TotalToolCalls    int         `json:"total_tool_calls"`
	TotalTurns        int         `json:"total_turns"`
	TotalAgents       int         `json:"total_agents"`
	Agents            []AgentInfo `json:"agents"`
	TurnAgents        []AgentInfo `json:"turn_agents"`
	LastUserTS        string      `json:"last_user_ts"`
	Model             string      `json:"model"`
	GitBranch         string      `json:"git_branch"`
}

// AgentInfo describes a sub-agent spawned during the session.
type AgentInfo struct {
	Description string `json:"desc"`
	Type        string `json:"type"`
	Name        string `json:"name"`
}

// Parse reads a transcript JSONL file and extracts statistics.
// Returns an empty Stats on any error (never nil).
func Parse(path string) *Stats {
	stats := &Stats{}
	if path == "" {
		return stats
	}

	f, err := os.Open(path)
	if err != nil {
		return stats
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Large lines possible — transcript entries can be big
	scanner.Buffer(make([]byte, 0, 1<<20), 10<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		var obj map[string]interface{}
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		// Extract git branch from top-level
		if gb, ok := obj["gitBranch"].(string); ok && gb != "" {
			stats.GitBranch = gb
		}

		recordType, _ := obj["type"].(string)
		ts, _ := obj["timestamp"].(string)

		switch recordType {
		case "user":
			stats.TotalTurns++
			// Only real user messages (not tool results): has userType=="external" and
			// no toolUseResult key. Tool results injected as 'user' type have toolUseResult.
			userType, _ := obj["userType"].(string)
			_, hasToolResult := obj["toolUseResult"]
			if userType == "external" && !hasToolResult {
				if ts != "" {
					stats.LastUserTS = ts
				}
				stats.TurnAgents = nil // reset per turn
			}

		case "assistant":
			msg, ok := obj["message"].(map[string]interface{})
			if !ok {
				continue
			}

			// Model
			if m, ok := msg["model"].(string); ok && m != "" {
				stats.Model = m
			}

			// Token usage
			if usage, ok := msg["usage"].(map[string]interface{}); ok {
				if tok, ok := usage["output_tokens"].(float64); ok {
					stats.TotalOutputTokens += int(tok)
				}
			}

			// Tool calls in content
			content, _ := msg["content"].([]interface{})
			for _, item := range content {
				c, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				if ctype, _ := c["type"].(string); ctype != "tool_use" {
					continue
				}
				stats.TotalToolCalls++

				// Track Agent tool calls specifically
				if name, _ := c["name"].(string); name == "Agent" {
					inp, _ := c["input"].(map[string]interface{})
					agent := AgentInfo{
						Description: strVal(inp, "description"),
						Type:        strVal(inp, "subagent_type"),
						Name:        strVal(inp, "name"),
					}
					stats.TotalAgents++
					stats.Agents = append(stats.Agents, agent)
					stats.TurnAgents = append(stats.TurnAgents, agent)
				}
			}
		}
	}

	return stats
}

func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
