package util

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	codeBlockStyle = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			PaddingLeft(1).
			Foreground(lipgloss.Color("#D1D5DB"))

	header1Style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F9FAFB")).PaddingTop(1)
	header2Style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB")).PaddingTop(1)
	header3Style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D1D5DB"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
	italicStyle  = lipgloss.NewStyle().Italic(true)
	codeStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#374151")).Foreground(lipgloss.Color("#FCA5A5")).Padding(0, 1)
	quoteStyle   = lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("#6B7280")).PaddingLeft(2).Foreground(lipgloss.Color("#9CA3AF")).Italic(true)
	linkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Underline(true)
	tableSep     = lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
	tableHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F9FAFB"))
	tableCell    = lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	listStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	listItem     = lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	hrStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
)

// RenderMarkdown converts markdown text to terminal-styled output.
func RenderMarkdown(md string, width int) string {
	if width <= 0 {
		width = 80
	}

	lines := strings.Split(md, "\n")
	var out strings.Builder
	var inCodeBlock bool
	var codeBuf strings.Builder
	var codeLang string
	var inTable bool
	var tableRows [][]string
	var tableAligns []string

	flushCodeBlock := func() {
		if codeBuf.Len() > 0 {
			code := codeBuf.String()
			// Strip trailing newline
			code = strings.TrimRight(code, "\n")
			if codeLang != "" {
				code = codeStyle.Render(" " + codeLang + " ") + "\n" + code
			}
			out.WriteString(codeBlockStyle.Render(code))
			out.WriteString("\n")
			codeBuf.Reset()
			codeLang = ""
		}
	}

	flushTable := func() {
		if len(tableRows) == 0 {
			return
		}

		// Calculate column widths
		colCount := 0
		for _, row := range tableRows {
			if len(row) > colCount {
				colCount = len(row)
			}
		}
		widths := make([]int, colCount)
		for _, row := range tableRows {
			for i, cell := range row {
				cellLen := lipgloss.Width(cell)
				if cellLen > widths[i] {
					widths[i] = cellLen
				}
			}
		}

		// Min column width
		for i := range widths {
			if widths[i] < 3 {
				widths[i] = 3
			}
		}

		totalWidth := 0
		for _, w := range widths {
			totalWidth += w + 3 // 3 for " │ " separator
		}

		// Render header
		if len(tableRows) > 0 {
			var headerParts []string
			for i, cell := range tableRows[0] {
				headerParts = append(headerParts, tableHeader.Render(padRight(cell, widths[i])))
			}
			out.WriteString(tableSep.Render("│ ") + strings.Join(headerParts, tableSep.Render(" │ ")) + tableSep.Render(" │"))
			out.WriteString("\n")

			// Separator
			var sepParts []string
			for i, w := range widths {
				align := "left"
				if i < len(tableAligns) {
					align = tableAligns[i]
				}
				line := strings.Repeat("─", w)
				switch align {
				case "right":
					line = strings.Repeat("─", w-1) + ":"
				case "center":
					line = ":" + strings.Repeat("─", w-2) + ":"
				}
				sepParts = append(sepParts, tableSep.Render(line))
			}
			out.WriteString(tableSep.Render("├─") + strings.Join(sepParts, tableSep.Render("─┼─")) + tableSep.Render("─┤"))
			out.WriteString("\n")

			// Data rows
			for _, row := range tableRows[1:] {
				var cells []string
				for i := 0; i < colCount; i++ {
					val := ""
					if i < len(row) {
						val = row[i]
					}
					cells = append(cells, tableCell.Render(padRight(val, widths[i])))
				}
				out.WriteString(tableSep.Render("│ ") + strings.Join(cells, tableSep.Render(" │ ")) + tableSep.Render(" │"))
				out.WriteString("\n")
			}
		}
		out.WriteString("\n")
		tableRows = nil
		tableAligns = nil
		inTable = false
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Code block toggle
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				flushCodeBlock()
				inCodeBlock = false
			} else {
				flushTable()
				inCodeBlock = true
				codeLang = strings.TrimPrefix(line, "```")
				codeLang = strings.TrimSpace(codeLang)
			}
			continue
		}

		if inCodeBlock {
			codeBuf.WriteString(line)
			codeBuf.WriteString("\n")
			continue
		}

		// Empty line: flush pending table
		if strings.TrimSpace(line) == "" {
			if inTable {
				flushTable()
			}
			out.WriteString("\n")
			continue
		}

		// Horizontal rule
		if line == "---" || line == "***" || line == "___" {
			flushTable()
			out.WriteString(hrStyle.Render(strings.Repeat("─", min(width, 80))))
			out.WriteString("\n")
			continue
		}

		// Table detection
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			cells := splitTableRow(line)
			if len(cells) >= 2 {
				if strings.Contains(line, "---") || strings.Contains(line, ":--") {
					// Alignment row
					for _, c := range cells {
						c = strings.TrimSpace(c)
						if len(c) >= 2 && c[0] == ':' && c[len(c)-1] == ':' {
							tableAligns = append(tableAligns, "center")
						} else if len(c) >= 1 && c[len(c)-1] == ':' {
							tableAligns = append(tableAligns, "right")
						} else {
							tableAligns = append(tableAligns, "left")
						}
					}
				} else if !inTable {
					// First data row - header may have already been set
					flushTable()
					inTable = true
					tableRows = append(tableRows, cells)
				} else {
					tableRows = append(tableRows, cells)
				}
				continue
			}
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			content := strings.TrimPrefix(line, "> ")
			content = applyInlineFormatting(content)
			out.WriteString(quoteStyle.Render(content))
			out.WriteString("\n")
			continue
		}

		// Headers
		if strings.HasPrefix(line, "### ") {
			flushTable()
			out.WriteString(header3Style.Render(applyInlineFormatting(strings.TrimPrefix(line, "### "))))
			out.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			flushTable()
			out.WriteString(header2Style.Render(applyInlineFormatting(strings.TrimPrefix(line, "## "))))
			out.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "# ") {
			flushTable()
			out.WriteString(header1Style.Render(applyInlineFormatting(strings.TrimPrefix(line, "# "))))
			out.WriteString("\n")
			continue
		}

		// Unordered list
		if match, _ := regexp.MatchString(`^(\s*)[-*+]\s`, line); match {
			flushTable()
			indent := len(line) - len(strings.TrimLeft(line, " "))
			content := strings.TrimLeft(line, " \t")
			content = strings.TrimPrefix(content, "- ")
			content = strings.TrimPrefix(content, "* ")
			content = strings.TrimPrefix(content, "+ ")
			prefix := strings.Repeat(" ", indent) + listStyle.Render("• ")
			out.WriteString(prefix + listItem.Render(applyInlineFormatting(content)))
			out.WriteString("\n")
			continue
		}

		// Ordered list
		if match, _ := regexp.MatchString(`^\s*\d+\.\s`, line); match {
			flushTable()
			re := regexp.MustCompile(`^(\s*)(\d+)\.\s(.*)`)
			parts := re.FindStringSubmatch(line)
			if len(parts) == 4 {
				prefix := parts[1] + listStyle.Render(parts[2]+". ")
				out.WriteString(prefix + listItem.Render(applyInlineFormatting(parts[3])))
				out.WriteString("\n")
				continue
			}
		}

		// Regular paragraph
		if inTable {
			flushTable()
		}
		out.WriteString(applyInlineFormatting(line))
		out.WriteString("\n")
	}

	// Flush remaining state
	flushCodeBlock()
	flushTable()

	return out.String()
}

// applyInlineFormatting handles bold, italic, code, and links in a single line.
func applyInlineFormatting(line string) string {
	// Bold **text** or __text__
	boldRe := regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	line = boldRe.ReplaceAllStringFunc(line, func(m string) string {
		inner := strings.TrimPrefix(m, "**")
		inner = strings.TrimPrefix(inner, "__")
		inner = strings.TrimSuffix(inner, "**")
		inner = strings.TrimSuffix(inner, "__")
		return boldStyle.Render(inner)
	})

	// Italic *text* or _text_ (not inside words starting with _)
	italicRe := regexp.MustCompile(`\*([^*\n]+)\*|(?:\s|^)_([^_\n]+)_(?:\s|$|[,.;:!?)])`)
	line = italicRe.ReplaceAllStringFunc(line, func(m string) string {
		inner := strings.Trim(m, "*_ ")
		return italicStyle.Render(inner)
	})

	// Inline code `text`
	codeRe := regexp.MustCompile("`([^`\n]+)`")
	line = codeRe.ReplaceAllStringFunc(line, func(m string) string {
		inner := strings.Trim(m, "`")
		return codeStyle.Render(inner)
	})

	// Links [text](url) → show text
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	line = linkRe.ReplaceAllString(line, linkStyle.Render("$1"))

	// Images ![alt](url) → show alt text
	imgRe := regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	line = imgRe.ReplaceAllString(line, italicStyle.Render("[img: $1]"))

	return line
}

// splitTableRow splits a table row into cells.
func splitTableRow(line string) []string {
	// Strip leading/trailing pipe
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	result := make([]string, len(cells))
	for i, c := range cells {
		result[i] = strings.TrimSpace(c)
	}
	return result
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
