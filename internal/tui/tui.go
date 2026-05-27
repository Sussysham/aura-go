package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"aura-go/internal/catalog"
	"aura-go/internal/config"
	"aura-go/internal/reader"
	"aura-go/internal/state"
	"aura-go/internal/util"
	"aura-go/internal/viewer"
)

func RunTUI(fd int, books []catalog.Book, libraryState *state.LibraryState, oldState *term.State, initialMsg string, initialColor string) {
	currentView := ViewExplorer
	activePanel := PanelLeft
	
	// Navigation indices
	selectedIndex := 0
	scrollOffset := 0
	subSelectedIndex := 0 

	// Console-Native Reader state variables
	var activeReaderBook catalog.Book
	var readerLines []string
	readerScrollIndex := 0
	activeReaderRawText := ""
	lastWrapWidth := 0
	
	// Reader search variables
	readerSearchQuery := ""
	isReaderSearching := false
	readerMatches := []int{}
	readerMatchIndex := -1
	
	// Reader session timer
	var readerStartTime time.Time
	showRawMarkdown := false

	searchQuery := ""
	isSearching := false
	statusMsg := initialMsg
	statusColor := initialColor
	
	var filteredBooks []catalog.Book
	filteredBooks = FilterBooks(books, libraryState, searchQuery)

	inputChan := make(chan []byte)
	go ReadInput(inputChan)

	for {
		// Fetch active terminal dimensions dynamically in the TUI loop
		width, height, err := term.GetSize(fd)
		if err != nil || width < 40 || height < 10 {
			width = 91
			height = 25
		}

		if currentView == ViewReader {
			wrapWidth := width - 12
			if wrapWidth < 40 {
				wrapWidth = 40
			}
			if wrapWidth > 120 {
				wrapWidth = 120 // clamp to comfort maximum
			}
			if wrapWidth != lastWrapWidth {
				readerLines = util.WrapText(activeReaderRawText, wrapWidth)
				lastWrapWidth = wrapWidth
				if readerScrollIndex >= len(readerLines) {
					readerScrollIndex = len(readerLines) - 1
					if readerScrollIndex < 0 { readerScrollIndex = 0 }
				}
			}
		}

		// --- AUTO VIEWPORT ALIGNER ---
		if currentView == ViewExplorer {
			visibleRows := height - 14
			if visibleRows < 5 {
				visibleRows = 5
			}
			if len(filteredBooks) == 0 {
				selectedIndex = 0
				scrollOffset = 0
			} else {
				if selectedIndex >= len(filteredBooks) {
					selectedIndex = len(filteredBooks) - 1
				}
				if selectedIndex < 0 {
					selectedIndex = 0
				}
				if selectedIndex < scrollOffset {
					scrollOffset = selectedIndex
				}
				if selectedIndex >= scrollOffset+visibleRows {
					scrollOffset = selectedIndex - (visibleRows - 1)
				}
				if scrollOffset < 0 {
					scrollOffset = 0
				}
			}
		}

		// Draw current view with clean Log Bar parameters
		switch currentView {
		case ViewExplorer:
			DrawExplorer(filteredBooks, selectedIndex, scrollOffset, isSearching, searchQuery, statusMsg, statusColor, libraryState, width, height)
		case ViewDashboard:
			DrawDashboard(books, libraryState, width, height)
		case ViewDuplicates:
			duplicates := catalog.GetDuplicateList(books)
			visibleDups := height - 14
			if visibleDups < 5 {
				visibleDups = 5
			}
			if len(duplicates) > 0 {
				if selectedIndex >= len(duplicates) { selectedIndex = len(duplicates) - 1 }
				if selectedIndex < 0 { selectedIndex = 0 }
				if selectedIndex < scrollOffset { scrollOffset = selectedIndex }
				if selectedIndex >= scrollOffset+visibleDups { scrollOffset = selectedIndex - (visibleDups - 1) }
			}
			DrawDuplicates(books, duplicates, selectedIndex, scrollOffset, subSelectedIndex, activePanel, statusMsg, statusColor, width, height)

		case ViewReader:
			DrawReader(activeReaderBook, readerLines, readerScrollIndex, readerSearchQuery, isReaderSearching, readerMatches, readerMatchIndex, libraryState, width, height, showRawMarkdown)
		}

		keys := <-inputChan
		if len(keys) == 0 {
			continue
		}

		// --- 1. GLOBAL PREEMPTIVE INTERRUPT HANDLERS ---
		if keys[0] == 3 {
			break
		}

		// Global ESC Key - Resets Explorer Search instantly or returns to main Explorer view
		if keys[0] == 27 && len(keys) == 1 {
			if currentView == ViewReader {
				if isReaderSearching {
					isReaderSearching = false
					readerSearchQuery = ""
					readerMatches = []int{}
					readerMatchIndex = -1
				} else {
					duration := int(time.Since(readerStartTime).Seconds())
					libraryState.ReadingTime[activeReaderBook.SHA1] += duration
					libraryState.BookProgress[activeReaderBook.SHA1] = readerScrollIndex
					if err := state.SaveState(config.StatePath, libraryState); err != nil {
						statusMsg = "⚠️ Save failed: " + err.Error()
						statusColor = config.ActiveTheme.ErrorSt
					} else {
						statusMsg = "Closed Reader. Logged " + util.FormatDuration(duration) + " reading session."
						statusColor = config.ActiveTheme.SuccessSt
					}
					
					currentView = ViewExplorer
					selectedIndex = 0
					scrollOffset = 0
					filteredBooks = FilterBooks(books, libraryState, searchQuery)
				}
			} else if isSearching || searchQuery != "" {
				isSearching = false
				searchQuery = ""
				filteredBooks = FilterBooks(books, libraryState, searchQuery)
				selectedIndex = 0
				scrollOffset = 0
				statusMsg = "Search cleared."
				statusColor = config.ActiveTheme.VioletSt
			} else if currentView != ViewExplorer {
				currentView = ViewExplorer
				activePanel = PanelLeft
				selectedIndex = 0
				scrollOffset = 0
				subSelectedIndex = 0
				filteredBooks = FilterBooks(books, libraryState, searchQuery)
				statusMsg = "Returned to Main Explorer."
				statusColor = config.ActiveTheme.SuccessSt
			}
			continue
		}

		// Quit app global
		if matchKey(keys, config.AppConfig.Keybindings.Quit) && !isSearching && currentView != ViewReader {
			break
		}

		// --- 2. INPUT HANDLING FOR TUI READER SEARCH MODE ---
		if currentView == ViewReader && isReaderSearching {
			if keys[0] == 13 || keys[0] == 10 { // Enter locks search
				isReaderSearching = false
				if len(readerMatches) > 0 {
					readerMatchIndex = 0
					readerScrollIndex = readerMatches[0]
				}
			} else if keys[0] == 127 || keys[0] == 8 { // Backspace
				if len(readerSearchQuery) > 0 {
					readerSearchQuery = readerSearchQuery[:len(readerSearchQuery)-1]
					readerMatches = SearchReaderText(readerLines, readerSearchQuery)
				}
			} else if keys[0] >= 32 && keys[0] <= 126 { // Typing
				readerSearchQuery += string(keys)
				readerMatches = SearchReaderText(readerLines, readerSearchQuery)
			}
			continue
		}

		// --- 3. INPUT HANDLING FOR GLOBAL EXPLORER SEARCH MODE ---
		if isSearching && currentView == ViewExplorer {
			if keys[0] == 13 || keys[0] == 10 { // Enter locks search
				isSearching = false
				statusMsg = "Search locked. Tap 'Esc' to clear search query."
				statusColor = config.ActiveTheme.SuccessSt
			} else if keys[0] == 127 || keys[0] == 8 { // Backspace
				if len(searchQuery) > 0 {
					searchQuery = searchQuery[:len(searchQuery)-1]
					filteredBooks = FilterBooks(books, libraryState, searchQuery)
					selectedIndex = 0
					scrollOffset = 0
				}
			} else if keys[0] >= 32 && keys[0] <= 126 { // Normal typing
				searchQuery += string(keys)
				filteredBooks = FilterBooks(books, libraryState, searchQuery)
				selectedIndex = 0
				scrollOffset = 0
			}
			continue
		}

		// Parse key actions
		var action string
		
		// Check for arrow keys (Standard ANSI)
		if len(keys) >= 3 && keys[0] == 27 && keys[1] == 91 {
			switch keys[2] {
			case 65: action = "UP"
			case 66: action = "DOWN"
			case 68: action = "LEFT"
			case 67: action = "RIGHT"
			}
		}

		// Check for Windows Console raw arrow scan keys
		if len(keys) >= 2 && keys[0] == 224 {
			switch keys[1] {
			case 72: action = "UP"
			case 80: action = "DOWN"
			case 75: action = "LEFT"
			case 77: action = "RIGHT"
			}
		}

		// Direct keyboard action fallbacks (j/k/h/l vim-style + Enter + Tab)
		if action == "" {
			switch keys[0] {
			case 'k': action = "UP"
			case 'j': action = "DOWN"
			case 'h': action = "LEFT"
			case 'l': action = "RIGHT"
			case 9:   action = "TAB" 
			case 13, 10: action = "ENTER"
			}
		}

		// --- 4. CONSOLE READER SCREEN LOGIC ---
		if currentView == ViewReader {
			visibleReaderLines := height - 10
			if visibleReaderLines < 5 {
				visibleReaderLines = 5
			}
			pageStep := visibleReaderLines - 2
			if pageStep < 3 {
				pageStep = 3
			}

			switch action {
			case "UP":
				if readerScrollIndex > 0 {
					readerScrollIndex--
				}
			case "DOWN":
				if readerScrollIndex < len(readerLines)-visibleReaderLines {
					readerScrollIndex++
				}
			}
			
			if action == "" {
				switch keys[0] {
				case 'u', 'b': // Page Up
					readerScrollIndex -= pageStep
					if readerScrollIndex < 0 { readerScrollIndex = 0 }
				case 'd', 'f', ' ': // Page Down
					readerScrollIndex += pageStep
					if readerScrollIndex > len(readerLines)-visibleReaderLines {
						readerScrollIndex = len(readerLines)-visibleReaderLines
						if readerScrollIndex < 0 { readerScrollIndex = 0 }
					}
				case '/': // Open reader find panel
					isReaderSearching = true
					readerSearchQuery = ""
					readerMatches = []int{}
					readerMatchIndex = -1
				case 'n': // Next match
					if len(readerMatches) > 0 && readerMatchIndex != -1 {
						readerMatchIndex = (readerMatchIndex + 1) % len(readerMatches)
						readerScrollIndex = readerMatches[readerMatchIndex]
					}
				case 'N': // Previous match
					if len(readerMatches) > 0 && readerMatchIndex != -1 {
						readerMatchIndex = (readerMatchIndex - 1 + len(readerMatches)) % len(readerMatches)
						readerScrollIndex = readerMatches[readerMatchIndex]
					}
				case 'e': // Edit margin notes
					_ = term.Restore(fd, oldState)
					fmt.Print("\033[H\033[2J") // Clear
					fmt.Printf("📖 \033[1;36mAdd Active Note for:\033[0m %s\n", activeReaderBook.Title)
					fmt.Println("\033[38;5;244mType your reading note/reflection below, then press Enter:\033[0m")
					fmt.Println("----------------------------------------------------------------------")
					if libraryState.Notes[activeReaderBook.SHA1] != "" {
						fmt.Printf("Current note: \033[1;33m%s\033[0m\n\n", libraryState.Notes[activeReaderBook.SHA1])
					}
					
					readerObj := bufio.NewReader(os.Stdin)
					fmt.Print("✍️  Enter Note: ")
					noteText, _ := readerObj.ReadString('\n')
					noteText = strings.TrimSpace(noteText)
					
					if noteText != "" {
						libraryState.Notes[activeReaderBook.SHA1] = noteText
						if err := state.SaveState(config.StatePath, libraryState); err != nil {
							fmt.Printf("\033[31m[ERROR] Failed to save note: %v\033[0m\n", err)
							fmt.Println("\nPress Enter to return...")
							_, _ = readerObj.ReadString('\n')
						}
					}

					newRawState, err := term.MakeRaw(fd)
					if err == nil {
						oldState = newRawState
					}
				case 'm': // Toggle Markdown syntax markers
					showRawMarkdown = !showRawMarkdown
				case 'q': // Exit reader
					duration := int(time.Since(readerStartTime).Seconds())
					libraryState.ReadingTime[activeReaderBook.SHA1] += duration
					libraryState.BookProgress[activeReaderBook.SHA1] = readerScrollIndex
					if err := state.SaveState(config.StatePath, libraryState); err != nil {
						statusMsg = "⚠️ Save failed: " + err.Error()
						statusColor = config.ActiveTheme.ErrorSt
					} else {
						statusMsg = "Closed Reader. Logged " + util.FormatDuration(duration) + " reading session."
						statusColor = config.ActiveTheme.SuccessSt
					}
					
					currentView = ViewExplorer
					selectedIndex = 0
					scrollOffset = 0
					filteredBooks = FilterBooks(books, libraryState, searchQuery)
				}
			}
			continue
		}

		// --- 5. EXPLORER VIEW LOGIC ---
		if currentView == ViewExplorer {
			switch action {
			case "UP":
				if selectedIndex > 0 {
					selectedIndex--
				}
			case "DOWN":
				if selectedIndex < len(filteredBooks)-1 {
					selectedIndex++
				}
			case "ENTER": // Open book in SumatraPDF & transition status
				if len(filteredBooks) > 0 {
					b := filteredBooks[selectedIndex]
					statusMsg = "Launching in SumatraPDF: " + b.Title + "..."
					statusColor = config.ActiveTheme.SuccessSt
					DrawExplorer(filteredBooks, selectedIndex, scrollOffset, isSearching, searchQuery, statusMsg, statusColor, libraryState, width, height)
					
					if libraryState.Statuses[b.SHA1] == "" || libraryState.Statuses[b.SHA1] == "Inbox" {
						libraryState.Statuses[b.SHA1] = "Reading"
						if err := state.SaveState(config.StatePath, libraryState); err != nil {
							statusMsg = "⚠️ Save failed: " + err.Error()
							statusColor = config.ActiveTheme.ErrorSt
						}
					}

					err := viewer.OpenBookInSumatra(b)
					if err != nil {
						statusMsg = "Sumatra Launch failed: " + err.Error()
						statusColor = config.ActiveTheme.ErrorSt
					} else {
						statusMsg = "Opened in SumatraPDF. Book status transitioned to 'Reading'."
						statusColor = config.ActiveTheme.SuccessSt
						filteredBooks = FilterBooks(books, libraryState, searchQuery)
					}
				}
			}

			// Letter commands
			if action == "" {
				charKey := string(keys[0])
				if matchKey(keys, config.AppConfig.Keybindings.Search) {
					isSearching = true
					searchQuery = ""
					statusMsg = "Search active. Type keywords, press 'Esc' to exit, 'Enter' to lock."
					statusColor = config.ActiveTheme.WarningSt
				} else if charKey == "v" {
					if len(filteredBooks) > 0 {
						b := filteredBooks[selectedIndex]
						statusMsg = "Extracting book text natively: " + b.FileName + "..."
						statusColor = config.ActiveTheme.WarningSt
						DrawExplorer(filteredBooks, selectedIndex, scrollOffset, isSearching, searchQuery, statusMsg, statusColor, libraryState, width, height)

						rawText, err := reader.ExtractBookRawText(b)
						if err != nil {
							statusMsg = "TUI Reader failed: " + err.Error() + " (Use SumatraPDF instead)"
							statusColor = config.ActiveTheme.ErrorSt
						} else {
							if libraryState.Statuses[b.SHA1] == "" || libraryState.Statuses[b.SHA1] == "Inbox" {
								libraryState.Statuses[b.SHA1] = "Reading"
								if err := state.SaveState(config.StatePath, libraryState); err != nil {
									statusMsg = "⚠️ Save failed: " + err.Error()
									statusColor = config.ActiveTheme.ErrorSt
								}
							}

							activeReaderBook = b
							activeReaderRawText = rawText
							lastWrapWidth = 0 // Force re-wrap calculation immediately
							readerStartTime = time.Now()
							
							readerScrollIndex = libraryState.BookProgress[b.SHA1]

							currentView = ViewReader
						}
					}
				} else if matchKey(keys, config.AppConfig.Keybindings.Dashboard) {
					currentView = ViewDashboard
				} else if matchKey(keys, config.AppConfig.Keybindings.Duplicates) {
					currentView = ViewDuplicates
					selectedIndex = 0
					scrollOffset = 0
					subSelectedIndex = 0
					activePanel = PanelLeft
					statusMsg = "Duplicates Deck loaded."
					statusColor = config.ActiveTheme.VioletSt
				} else if matchKey(keys, config.AppConfig.Keybindings.Ingest) {
					_ = term.Restore(fd, oldState)
					fmt.Print("\033[H\033[2J") // Clear
					fmt.Println("📖 \033[1;36mAURA PORTABLE INGESTION ENGINE\033[0m")
					fmt.Println("----------------------------------------------------------------------")
					fmt.Println("Enter the absolute directory path to scan for ebooks/documents")
					fmt.Println("(or press Enter to scan the current working directory):")
					fmt.Println()
					
					readerObj := bufio.NewReader(os.Stdin)
					fmt.Print("📂 Path: ")
					scanDir, _ := readerObj.ReadString('\n')
					scanDir = strings.TrimSpace(scanDir)
					
					if scanDir == "" {
						scanDir, _ = os.Getwd()
					}
					
					fmt.Printf("\n🔍 Scanning directory recursively: %s...\n", scanDir)
					newBooks, err := catalog.ScanDirectoryForBooks(scanDir)
					if err != nil {
						fmt.Printf("\033[31m[ERROR] Scan failed: %v\033[0m\n", err)
						fmt.Println("\nPress Enter to return to TUI...")
						_, _ = readerObj.ReadString('\n')
					} else {
						err = catalog.AppendBooksToCatalog(config.CSVPath, newBooks)
						if err != nil {
							fmt.Printf("\033[31m[ERROR] Ingestion failed: %v\033[0m\n", err)
						} else {
							fmt.Printf("\n\033[32m✔ [SUCCESS] Successfully scanned & parsed %d books!\033[0m\n", len(newBooks))
							books, err = catalog.LoadBooks(config.CSVPath)
							if err != nil {
								fmt.Printf("\033[31m[ERROR] Reloading catalog failed: %v\033[0m\n", err)
							}
						}
						fmt.Println("\nPress Enter to reload TUI...")
						_, _ = readerObj.ReadString('\n')
					}

					newRawState, err := term.MakeRaw(fd)
					if err == nil {
						oldState = newRawState
					}
					filteredBooks = FilterBooks(books, libraryState, searchQuery)
					selectedIndex = 0
					scrollOffset = 0
				} else if matchKey(keys, config.AppConfig.Keybindings.ThemeCycle) {
					currentTheme := strings.ToLower(config.AppConfig.Theme.Name)
					themes := []string{"midnight", "nord", "dracula", "solarized", "gruvbox", "contrast", "custom"}
					nextIdx := 0
					for idx, th := range themes {
						if th == currentTheme {
							nextIdx = (idx + 1) % len(themes)
							break
						}
					}
					nextTheme := themes[nextIdx]
					config.AppConfig.Theme.Name = nextTheme
					config.LoadTheme(nextTheme, config.AppConfig.Theme.Custom)
					statusMsg = "Cycled theme to: " + strings.Title(nextTheme)
					statusColor = config.ActiveTheme.SuccessSt
				} else if charKey == "w" {
					if len(filteredBooks) > 0 {
						b := filteredBooks[selectedIndex]
						currStatus := libraryState.Statuses[b.SHA1]
						if currStatus == "" {
							currStatus = "Inbox"
						}

						var nextStatus string
						switch currStatus {
						case "Inbox":
							wipCount := 0
							for _, s := range libraryState.Statuses {
								if s == "Reading" {
									wipCount++
								}
							}
							if wipCount >= config.AppConfig.General.WIPLimit {
								statusMsg = fmt.Sprintf("⛔ WIP Limit Exceeded! Archive or finish one of your active reading books first! (Limit: %d)", config.AppConfig.General.WIPLimit)
								statusColor = config.ActiveTheme.ErrorSt
								continue
							}
							nextStatus = "Reading"
						case "Reading":
							nextStatus = "Read"
						case "Read":
							nextStatus = "Reference"
						default:
							nextStatus = "Inbox"
						}

						libraryState.Statuses[b.SHA1] = nextStatus
						if err := state.SaveState(config.StatePath, libraryState); err != nil {
							statusMsg = "⚠️ Status saved in memory, but disk save failed: " + err.Error()
							statusColor = config.ActiveTheme.ErrorSt
						} else {
							statusMsg = "Status updated: " + b.Title + " -> " + nextStatus
							statusColor = config.ActiveTheme.SuccessSt
						}
						filteredBooks = FilterBooks(books, libraryState, searchQuery)
					}
				}
			}
		}

		// --- 7. DUPLICATES VIEW DECK LOGIC ---
		if currentView == ViewDuplicates {
			duplicates := catalog.GetDuplicateList(books)
			
			switch action {
			case "UP":
				if activePanel == PanelLeft {
					if selectedIndex > 0 {
						selectedIndex--
						subSelectedIndex = 0
					}
				} else {
					if subSelectedIndex > 0 {
						subSelectedIndex--
					}
				}
			case "DOWN":
				if activePanel == PanelLeft {
					if selectedIndex < len(duplicates)-1 {
						selectedIndex++
						subSelectedIndex = 0
					}
				} else {
					activeGrp := duplicates[selectedIndex]
					groupBooks := catalog.GetBooksByGroup(books, activeGrp)
					if subSelectedIndex < len(groupBooks)-1 {
						subSelectedIndex++
					}
				}
			case "RIGHT", "TAB":
				if activePanel == PanelLeft && len(duplicates) > 0 {
					activePanel = PanelRight
					subSelectedIndex = 0
					statusMsg = "Focused Right Panel."
					statusColor = config.ActiveTheme.SuccessSt
				}
			case "LEFT":
				if activePanel == PanelRight {
					activePanel = PanelLeft
					statusMsg = "Focused Left Panel."
					statusColor = config.ActiveTheme.VioletSt
				}
			case "ENTER":
				if activePanel == PanelRight && len(duplicates) > 0 {
					activeGrp := duplicates[selectedIndex]
					groupBooks := catalog.GetBooksByGroup(books, activeGrp)
					if subSelectedIndex < len(groupBooks) {
						target := groupBooks[subSelectedIndex]
						statusMsg = "Opening to visually compare in SumatraPDF: " + target.FileName + "..."
						statusColor = config.ActiveTheme.SuccessSt
						DrawDuplicates(books, duplicates, selectedIndex, scrollOffset, subSelectedIndex, activePanel, statusMsg, statusColor, width, height)
						
						_ = viewer.OpenBookInSumatra(target)
					}
				}
			}

			// Pruning and Panel switching actions
			if activePanel == PanelRight {
				if keys[0] == 'd' && len(duplicates) > 0 {
					activeGrp := duplicates[selectedIndex]
					groupBooks := catalog.GetBooksByGroup(books, activeGrp)
					if len(groupBooks) > 1 && subSelectedIndex < len(groupBooks) {
						target := groupBooks[subSelectedIndex]
						
						statusMsg = "Deleting duplicate copy..."
						statusColor = config.ActiveTheme.WarningSt
						DrawDuplicates(books, duplicates, selectedIndex, scrollOffset, subSelectedIndex, activePanel, statusMsg, statusColor, width, height)

						err := catalog.DeleteDuplicateSecurely(target)
						if err != nil {
							statusMsg = "Delete failed: " + err.Error()
							statusColor = config.ActiveTheme.ErrorSt
						} else {
							books = catalog.RemoveBookFromMem(books, target.FullPath)
							statusMsg = "🧹 [DELETED] Duplicate pruned successfully!"
							statusColor = config.ActiveTheme.SuccessSt
							subSelectedIndex = 0
							activePanel = PanelLeft
						}
					}
				}
			} else {
				// activePanel == PanelLeft
				switch keys[0] {
				case 'd':
					currentView = ViewDashboard
					selectedIndex = 0
					scrollOffset = 0
					statusMsg = "Dashboard loaded."
					statusColor = config.ActiveTheme.SuccessSt
				}
			}
		}

		// --- 9. DASHBOARD VIEW LOGIC ---
		if currentView == ViewDashboard {
			if action == "" {
				switch keys[0] {
				case 'u':
					currentView = ViewDuplicates
					selectedIndex = 0
					scrollOffset = 0
					subSelectedIndex = 0
					activePanel = PanelLeft
					statusMsg = "Duplicates Deck loaded from Dashboard."
					statusColor = config.ActiveTheme.VioletSt
				case 'd':
					currentView = ViewExplorer
					selectedIndex = 0
					scrollOffset = 0
					statusMsg = "Returned to main Explorer."
					statusColor = config.ActiveTheme.SuccessSt
				}
			}
		}
	}
}

// UI View states
type ViewState int
const (
	ViewExplorer ViewState = iota
	ViewDashboard
	ViewDuplicates
	ViewReader // Console-Native Ebook Reader Screen
)

func matchKey(keys []byte, cfgKey string) bool {
	return len(keys) == 1 && cfgKey != "" && keys[0] == cfgKey[0]
}
