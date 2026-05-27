package tui

import (
	"fmt"

	"aura-go/internal/catalog"
	"aura-go/internal/config"
	"aura-go/internal/state"
)

func DrawExplorer(books []catalog.Book, sel, offset int, isSearching bool, query, statusMsg, statusColor string, state *state.LibraryState, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	DrawLine(width, "=", config.ColorViolet)
	fmt.Printf("%s                             AURA COMPARED TUI ENGINE (GO)                                 \033[0m\n", config.ColorHeader)
	DrawLine(width, "=", config.ColorViolet)
	
	// WIP Active Focus Deck
	wipSlots := []string{}
	for _, b := range books {
		if state.Statuses[b.SHA1] == "Reading" {
			wipSlots = append(wipSlots, b.Title)
		}
	}
	fmt.Printf("⚡ %s[WIP FOCUS DECK (Max 2)]\033[0m ", config.ColorSuccess)
	if len(wipSlots) == 0 {
		fmt.Println("\033[37mEmpty. (Highlight a book and press 'w' to set Reading Focus!)\033[0m")
	} else {
		for i, w := range wipSlots {
			if len(w) > 30 {
				w = w[:27] + "..."
			}
			fmt.Printf("%sSlot %d: %s\033[0m", config.ColorWarning, i+1, w)
			if i < len(wipSlots)-1 {
				fmt.Print("  |  ")
			}
		}
		fmt.Println()
	}
	DrawLine(width, "-", config.ColorMuted)

	titleWidth := width - 48
	if titleWidth < 20 {
		titleWidth = 20
	}
	
	headerFormat := fmt.Sprintf("\033[1;37m%%-3s | %%-%ds | %%-18s | %%-9s | %%-6s\033[0m\n", titleWidth)
	fmt.Printf(headerFormat, "Idx", "Book Title", "Author", "Status", "Format")
	DrawLine(width, "-", config.ColorMuted)

	visibleRows := height - 14
	if visibleRows < 5 {
		visibleRows = 5
	}

	if len(books) == 0 {
		fmt.Println()
		fmt.Printf("    %sWelcome to Aura TUI! Your Native Portable Library is Ready.\033[0m\n", config.ColorHeader)
		fmt.Println()
		fmt.Println("    To get started, you can automatically index your ebook files:")
		fmt.Printf("    1. Press %s'i'\033[0m to enter the directory auto-discovery scanner.\n", config.ColorSuccess)
		fmt.Println("    2. Provide the absolute directory path of your book collection.")
		fmt.Println("    3. Aura will recursively crawl, clean, fingerprint, and load them instantly!")
		fmt.Println()
		fmt.Printf("    Alternatively, you can drop a catalog file named %s'books_catalog.csv'\033[0m\n", config.ColorWarning)
		fmt.Println("    directly in this executable's directory to load your assets.")
		fmt.Println()
	} else {
		end := offset + visibleRows
		if end > len(books) {
			end = len(books)
		}

		for idx := offset; idx < end; idx++ {
			b := books[idx]
			title := b.Title
			if len(title) > titleWidth {
				title = title[:titleWidth-3] + "..."
			}
			author := b.Author
			if len(author) > 18 {
				author = author[:15] + "..."
			}
			
			bStatus := state.Statuses[b.SHA1]
			if bStatus == "" {
				bStatus = "Inbox"
			}

			statusColorCode := "\033[37m"
			switch bStatus {
			case "Reading":
				statusColorCode = config.ColorWarning
			case "Read":
				statusColorCode = config.ColorSuccess
			case "Reference":
				statusColorCode = config.ColorViolet
			}

			if idx == sel {
				rowFormat := fmt.Sprintf("%s%%03d | %%-%ds | %%-18s | %%-9s | %%-6s\033[0m\n", config.ColorSelect, titleWidth)
				fmt.Printf(rowFormat, idx+1, title, author, bStatus, b.Format)
			} else {
				rowFormat := fmt.Sprintf("%s%%03d\033[0m | %%-%ds | %%-18s | %%s%%-9s\033[0m | %%-6s\n", config.ColorAccent, titleWidth)
				fmt.Printf(rowFormat, idx+1, title, author, statusColorCode, bStatus, b.Format)
			}
		}
	}

	DrawLine(width, "-", config.ColorMuted)

	// Search bar rendering
	if isSearching {
		fmt.Printf("🔍 %sSEARCH FILTER:%s %s▮\n", config.ColorWarning, "\033[0m", query)
	} else if query != "" {
		fmt.Printf("🔍 %sACTIVE FILTER:%s %s (Press 'Esc' to clear or change)\n", config.ColorHeader, "\033[0m", query)
	} else {
		fmt.Println("🔍 Type query directly in search mode by pressing 's'")
	}

	DrawLine(width, "=", config.ColorMuted)
	
	// v5 HUD - Log Bar + Dynamic Shortcut Guide
	fmt.Printf("%s🔔 LOG: %s\033[0m\n", statusColor, statusMsg)
	if isSearching {
		fmt.Println("\033[1;35m💡 GUIDE: [Type] to filter | [Backspace] Delete char | [Enter] Lock filter | [Esc] Cancel search\033[0m")
	} else if query != "" {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll | [s] Edit Search | [Esc] Clear Search | [Enter] SumatraPDF | [v] TUI Reader | [w] Status | [i] Ingest Scan | [d] Stats | [u] Dups | [q] Exit\033[0m")
	} else {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll | [s] Search | [Enter] SumatraPDF | [v] TUI Reader | [w] Status Toggle | [i] Ingest Scan | [d] Stats | [u] Dups | [q] Exit\033[0m")
	}
}
