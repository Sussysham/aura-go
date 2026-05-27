package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"aura-go/internal/catalog"
	"aura-go/internal/config"
	"aura-go/internal/reader"
	"aura-go/internal/state"
)

func DrawReader(b catalog.Book, lines []string, scrollIdx int, findQuery string, isSearching bool, matches []int, matchIdx int, state *state.LibraryState, width, height int, showRawMarkdown bool) {
	fmt.Print("\033[H\033[2J") // Clear screen
	DrawLine(width, "=", config.ActiveTheme.Violet)
	
	titleDisplay := b.Title
	if len(titleDisplay) > 50 { titleDisplay = titleDisplay[:47] + "..." }
	
	visibleLines := height - 10
	if visibleLines < 5 {
		visibleLines = 5
	}

	percentage := 0.0
	if len(lines) > visibleLines {
		percentage = (float64(scrollIdx) / float64(len(lines)-visibleLines)) * 100.0
		if percentage > 100.0 { percentage = 100.0 }
	}
	
	fmt.Printf("📖 \033[1;36m%-50s\033[0m | %sProgress: %3.1f%%\033[0m (Line %d/%d)\n", titleDisplay, config.ActiveTheme.Warning, percentage, scrollIdx+1, len(lines))
	DrawLine(width, "=", config.ActiveTheme.Violet)

	if len(lines) == 0 {
		fmt.Print("\n                    \033[31m[EMPTY] No text could be extracted from this book.\033[0m\n\n")
	} else {
		end := scrollIdx + visibleLines
		if end > len(lines) {
			end = len(lines)
		}

		isMarkdown := strings.ToLower(filepath.Ext(b.FullPath)) == ".md"
		inCodeBlock := false

		for idx := scrollIdx; idx < end; idx++ {
			lineText := lines[idx]
			
			// Highlight search queries in glowing yellow in the TUI reader!
			if findQuery != "" {
				re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(findQuery))
				if err == nil {
					lineText = re.ReplaceAllString(lineText, "\033[7;33m$0\033[0m\033[37m")
				}
			}

			// Render Markdown format dynamically
			if isMarkdown {
				trimmed := strings.TrimSpace(lineText)
				if strings.HasPrefix(trimmed, "```") {
					inCodeBlock = !inCodeBlock
					lineText = config.ActiveTheme.Muted + "════════════════════════════════════════" + "\033[0m"
				} else if inCodeBlock {
					lineText = config.ActiveTheme.Muted + "  " + lineText + "\033[0m"
				} else {
					lineText = reader.FormatMarkdownLine(lineText, config.ActiveTheme, showRawMarkdown)
				}
			}

			// Render highlighted match lines distinctly
			isMatchLine := false
			for _, mIdx := range matches {
				if mIdx == idx {
					isMatchLine = true
					break
				}
			}

			if isMatchLine {
				fmt.Printf(" %s%04d\033[0m | \033[37m%s\033[0m\n", config.ActiveTheme.Success, idx+1, lineText)
			} else {
				fmt.Printf(" %s%04d\033[0m | %s\n", config.ActiveTheme.Muted, idx+1, lineText)
			}
		}
	}

	DrawLine(width, "=", config.ActiveTheme.Violet)
	
	// Active Margin Note display at bottom of card
	noteText := state.Notes[b.SHA1]
	if noteText == "" {
		noteText = "No active margin notes. Press 'e' to add your first reflection/summary!"
	} else {
		if len(noteText) > width-10 && width > 15 {
			noteText = noteText[:width-13] + "..."
		}
		noteText = "✍️  Margin Note: \"" + noteText + "\""
	}
	fmt.Printf("%s%s\033[0m\n", config.ActiveTheme.Warning, noteText)
	DrawLine(width, "-", config.ActiveTheme.Muted)

	// Find panel rendering
	if findQuery != "" {
		matchStr := "0 matches"
		if len(matches) > 0 {
			matchStr = fmt.Sprintf("%d/%d matches", matchIdx+1, len(matches))
		}
		fmt.Printf("🔍 %sFIND IN BOOK:%s %s▮ (%s) | Press 'n'/'N' to cycle\n", config.ActiveTheme.Warning, "\033[0m", findQuery, matchStr)
	} else {
		fmt.Println("🔍 Press '/' to Find text in this book | Press 'e' to Edit Note")
	}

	DrawLine(width, "=", config.ActiveTheme.Violet)
	// v5 HUD for Reader
	if isSearching {
		fmt.Println("\033[1;35m💡 GUIDE: [Type] to find | [Backspace] Delete char | [Enter] Go to First Match | [Esc] Cancel search\033[0m")
	} else {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll | [Space/d] Page Down | [u/b] Page Up | [/] Find Text | [n/N] Next Match | [e] Edit Note | [m] Toggle Markdown | [Esc/q] Exit Reader\033[0m")
	}
}
