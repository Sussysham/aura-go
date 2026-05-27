package reader

import (
	"regexp"
	"strings"

	"aura-go/internal/config"
)

func FormatMarkdownLine(line string, theme *config.Theme, showRaw bool) string {
	// 1. Headers
	if strings.HasPrefix(line, "# ") {
		content := strings.TrimPrefix(line, "# ")
		if showRaw {
			return theme.Header + "\033[1m# " + content + "\033[0m"
		}
		return theme.Header + "\033[1m" + content + "\033[0m"
	}
	if strings.HasPrefix(line, "## ") {
		content := strings.TrimPrefix(line, "## ")
		if showRaw {
			return theme.Accent + "\033[1m## " + content + "\033[0m"
		}
		return theme.Accent + "\033[1m" + content + "\033[0m"
	}
	if strings.HasPrefix(line, "### ") {
		content := strings.TrimPrefix(line, "### ")
		if showRaw {
			return "\033[1m### " + content + "\033[0m"
		}
		return "\033[1m" + content + "\033[0m"
	}

	// 2. Blockquotes
	if strings.HasPrefix(line, "> ") {
		content := strings.TrimPrefix(line, "> ")
		if showRaw {
			return theme.Muted + "> " + content + "\033[0m"
		}
		return theme.Muted + "│ " + content + "\033[0m"
	}

	// 3. Lists / Bullets
	if strings.HasPrefix(line, "- ") {
		content := strings.TrimPrefix(line, "- ")
		if showRaw {
			return theme.Accent + "- \033[0m" + content
		}
		return "  " + theme.Accent + "•\033[0m " + content
	}
	if strings.HasPrefix(line, "* ") {
		content := strings.TrimPrefix(line, "* ")
		if showRaw {
			return theme.Accent + "* \033[0m" + content
		}
		return "  " + theme.Accent + "•\033[0m " + content
	}

	// 4. Inline elements: Bold, Italic, Code
	formatted := line

	if showRaw {
		// Keep markers, but colorize the whole block nicely
		// Bold: **text** -> **\033[1mtext\033[22m**
		formatted = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(formatted, theme.Violet+"**\033[1m$1\033[22m**\033[0m")
		formatted = regexp.MustCompile(`__(.*?)__`).ReplaceAllString(formatted, theme.Violet+"__\033[1m$1\033[22m__\033[0m")

		// Italic: *text* -> *\033[3mtext\033[23m*
		formatted = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(formatted, theme.Violet+"*\033[3m$1\033[23m*\033[0m")
		formatted = regexp.MustCompile(`_(.*?)_`).ReplaceAllString(formatted, theme.Violet+"_\033[3m$1\033[23m_\033[0m")

		// Inline Code: `code` -> `\033[36mcode\033[0m`
		formatted = regexp.MustCompile("`(.*?)`").ReplaceAllString(formatted, "`"+theme.Accent+"$1\033[0m`")
	} else {
		// Hide markers, display pure style
		// Bold: **text** -> \033[1mtext\033[22m
		formatted = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(formatted, "\033[1m$1\033[22m")
		formatted = regexp.MustCompile(`__(.*?)__`).ReplaceAllString(formatted, "\033[1m$1\033[22m")

		// Italic: *text* -> \033[3mtext\033[23m
		formatted = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(formatted, "\033[3m$1\033[23m")
		formatted = regexp.MustCompile(`_(.*?)_`).ReplaceAllString(formatted, "\033[3m$1\033[23m")

		// Inline Code: `code` -> color block
		formatted = regexp.MustCompile("`(.*?)`").ReplaceAllString(formatted, theme.Accent+"$1\033[0m")
	}

	return formatted
}
