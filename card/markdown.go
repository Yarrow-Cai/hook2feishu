package card

import (
	"regexp"
	"strings"
)

// CleanMarkdown converts standard Markdown to the Feishu card-compatible subset.
//
// Feishu card markdown supports:
//
//	**bold**, *italic*, ~~strike~~, [link](url), `inline code`,
//	<font color='red/green/grey'>colored text</font>
//
// Does NOT support: # headings, tables, code blocks, ordered/unordered lists, blockquotes.
// We convert unsupported elements to their best approximation.
func CleanMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inCodeBlock := false

	hdrRe := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	tableSepRe := regexp.MustCompile(`^\s*\|[\s:]*-+[\s:]*(?:\|[\s:]*-+[\s:]*)*\|\s*$`)
	tableRowRe := regexp.MustCompile(`^\s*\|(.+)\|\s*$`)
	listRe := regexp.MustCompile(`^(\s*)[-*]\s+(.+)$`)

	for _, line := range lines {
		// Code block toggle
		if strings.TrimSpace(line) == "```" {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			s := strings.TrimRight(line, " \t")
			if s != "" {
				out = append(out, "`"+s+"`")
			} else {
				out = append(out, "")
			}
			continue
		}

		// Headings → bold (with color for emphasis)
		if m := hdrRe.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			title := strings.TrimSpace(m[2])
			if level <= 2 {
				out = append(out, "<font color='green'>**"+title+"**</font>")
			} else {
				out = append(out, "**"+title+"**")
			}
			continue
		}

		// Table separator → skip
		if tableSepRe.MatchString(line) {
			continue
		}

		// Table rows: | A | B | → A  B
		if m := tableRowRe.FindStringSubmatch(line); m != nil {
			cells := strings.Split(m[1], "|")
			var trimmed []string
			for _, c := range cells {
				trimmed = append(trimmed, strings.TrimSpace(c))
			}
			out = append(out, strings.Join(trimmed, "  "))
			continue
		}

		// Blockquotes → italic
		if strings.HasPrefix(line, "> ") {
			out = append(out, "*"+line[2:]+"*")
			continue
		}

		// Unordered list → bullet
		if m := listRe.FindStringSubmatch(line); m != nil {
			indent := strings.Repeat("  ", len(m[1])/2)
			out = append(out, indent+"• "+m[2])
			continue
		}

		// Everything else: keep as-is
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}
