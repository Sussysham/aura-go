package tui

import (
	"fmt"

	"aura-go/internal/catalog"
	"aura-go/internal/config"
)

// Active focus panels for dual-panel views (Duplicates)
type FocusPanel int
const (
	PanelLeft FocusPanel = iota
	PanelRight
)

func DrawDuplicates(books []catalog.Book, duplicates []string, sel, offset, subSel int, activePanel FocusPanel, statusMsg, statusColor string, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	DrawLine(width, "=", config.ColorViolet)
	fmt.Printf("%s                          AURA HARDENED DEDUPLICATION ENGINE                               \033[0m\n", config.ColorError)
	DrawLine(width, "=", config.ColorViolet)

	visibleDups := height - 14
	if visibleDups < 5 {
		visibleDups = 5
	}

	if len(duplicates) == 0 {
		fmt.Printf("\n                    %s✔ [EXCELLENT] Zero duplicate files found on disk!\033[0m\n\n", config.ColorSuccess)
	} else {
		fmt.Println("LEFT: Duplicate Groups list  |  RIGHT: Specific duplicate files (Press Tab to toggle)")
		DrawLine(width, "-", config.ColorMuted)

		end := offset + visibleDups
		if end > len(duplicates) {
			end = len(duplicates)
		}

		for idx := offset; idx < end; idx++ {
			grp := duplicates[idx]
			groupBooks := catalog.GetBooksByGroup(books, grp)
			title := "Unknown"
			if len(groupBooks) > 0 {
				title = groupBooks[0].Title
			}
			if len(title) > 40 {
				title = title[:37] + "..."
			}

			if idx == sel {
				panelIndicator := "◀"
				if activePanel == PanelRight {
					panelIndicator = " "
				}
				fmt.Printf("%sGroup %02d %s [DUP-ID: %-5s] %-40s | %d copies\033[0m\n", config.ColorSelectRed, idx+1, panelIndicator, grp, title, len(groupBooks))
			} else {
				fmt.Printf("%sGroup %02d\033[0m   [DUP-ID: %-5s] %-40s | %d copies\n", config.ColorError, idx+1, grp, title, len(groupBooks))
			}
		}

		DrawLine(width, "-", config.ColorMuted)

		if sel < len(duplicates) {
			activeGrp := duplicates[sel]
			groupBooks := catalog.GetBooksByGroup(books, activeGrp)
			
			rightHeaderColor := "\033[1;33m"
			if activePanel == PanelRight {
				rightHeaderColor = "\033[1;32m▶ "
			}
			fmt.Printf("%sCompare Duplicate Files (Group DUP-ID: %s):\033[0m\n", rightHeaderColor, activeGrp)

			for i, gb := range groupBooks {
				actionText := "\033[1;34m[ORIGINAL FILE]\033[0m"
				if i > 0 {
					actionText = "\033[1;31m[REDUNDANT COPY]\033[0m"
				}

				pathDisplay := gb.FullPath
				if len(pathDisplay) > width-15 && width > 20 {
					pathDisplay = "..." + pathDisplay[len(pathDisplay)-(width-18):]
				}

				rowContent := fmt.Sprintf("  [%d] %s Size: %.2f MB | Modified: %s\n      Path: %s", i+1, actionText, gb.SizeMB, gb.Modified, pathDisplay)
				
				if activePanel == PanelRight && i == subSel {
					fmt.Printf("%s%s\033[0m\n", config.ColorSelectGrn, rowContent)
				} else {
					fmt.Printf("%s\n", rowContent)
				}
			}
		}
	}

	DrawLine(width, "=", config.ColorMuted)
	
	// v5 HUD for Duplicates (Dynamic key guide based on panel focus!)
	fmt.Printf("%s🔔 LOG: %s\033[0m\n", statusColor, statusMsg)
	if activePanel == PanelLeft {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll Groups | [Tab/→/l] Focus Copies | [Esc] Back to Explorer | [d] Dashboard | [q] Exit App\033[0m")
	} else {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll Copies | [Enter] SumatraPDF | [d] PRUNE (Delete) Copy | [←/h] Focus Groups | [Esc] Back to Explorer\033[0m")
	}
}
