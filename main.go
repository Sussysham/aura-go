package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// Book represents ebook metadata ingested from the catalog CSV
type Book struct {
	Title             string
	Author            string
	Format            string
	SizeMB            float64
	LocationCategory  string
	DuplicateGroup    string
	FilenameGibberish string
	SHA1              string
	FileName          string
	FullPath          string
	Modified          string
	Tags              string
}

// LibraryState stores reading statuses, scroll positions, notes, and reading time
type LibraryState struct {
	Statuses     map[string]string `json:"statuses"`
	Notes        map[string]string `json:"notes"`
	BookProgress map[string]int    `json:"book_progress"` // Maps Book SHA1 to last scroll line index
	ReadingTime  map[string]int    `json:"reading_time"`   // Maps Book SHA1 to total seconds read
}

var (
	csvPath   string
	statePath string
)

func initPaths() {
	originalCSV := `D:\Audit and revam\_inventory\books_catalog.csv`
	originalState := `D:\Audit and revam\aura-go\library_state.json`
	if _, err := os.Stat(originalCSV); err == nil {
		csvPath = originalCSV
		statePath = originalState
		return
	}
	
	currDir, err := os.Getwd()
	if err == nil {
		csvPath = filepath.Join(currDir, "books_catalog.csv")
		statePath = filepath.Join(currDir, "library_state.json")
	} else {
		csvPath = "books_catalog.csv"
		statePath = "library_state.json"
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

// Active focus panels for dual-panel views (Duplicates)
type FocusPanel int
const (
	PanelLeft FocusPanel = iota
	PanelRight
)

func main() {
	initPaths()
	state := loadState(statePath)
	books, err := loadBooks(csvPath)
	if err != nil {
		books = []Book{}
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Printf("[ERROR] Failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = term.Restore(fd, oldState)
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("Aura TUI closed successfully.")
	}()

	runTUI(fd, books, state, oldState)
}

func loadState(path string) *LibraryState {
	state := &LibraryState{
		Statuses:     make(map[string]string),
		Notes:        make(map[string]string),
		BookProgress: make(map[string]int),
		ReadingTime:  make(map[string]int),
	}
	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		_ = json.NewDecoder(file).Decode(state)
	}
	return state
}

func saveState(path string, state *LibraryState) {
	file, err := os.Create(path)
	if err == nil {
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(state)
	}
}

func loadBooks(path string) ([]Book, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	var books []Book
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		sizeMB, _ := strconv.ParseFloat(record[headerMap["SizeMB"]], 64)
		
		tagsVal := ""
		if idx, exists := headerMap["Tags"]; exists {
			tagsVal = record[idx]
		}

		books = append(books, Book{
			Title:             record[headerMap["Title"]],
			Author:            record[headerMap["Author"]],
			Format:            record[headerMap["Format"]],
			SizeMB:            sizeMB,
			LocationCategory:  record[headerMap["LocationCategory"]],
			DuplicateGroup:    record[headerMap["DuplicateGroup"]],
			FilenameGibberish: record[headerMap["FilenameGibberish"]],
			SHA1:              record[headerMap["SHA1"]],
			FileName:          record[headerMap["FileName"]],
			FullPath:          record[headerMap["FullPath"]],
			Modified:          record[headerMap["Modified"]],
			Tags:              tagsVal,
		})
	}
	return books, nil
}

func runTUI(fd int, books []Book, state *LibraryState, oldState *term.State) {
	currentView := ViewExplorer
	activePanel := PanelLeft
	
	// Navigation indices
	selectedIndex := 0
	scrollOffset := 0
	subSelectedIndex := 0 

	// Console-Native Reader state variables
	var activeReaderBook Book
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

	searchQuery := ""
	isSearching := false
	statusMsg := "Ready."
	statusColor := "\033[32m" // Green
	
	var filteredBooks []Book
	filteredBooks = filterBooks(books, state, searchQuery)

	inputChan := make(chan []byte)
	go readInput(inputChan)

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
				readerLines = wrapText(activeReaderRawText, wrapWidth)
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
			drawExplorer(filteredBooks, selectedIndex, scrollOffset, isSearching, searchQuery, statusMsg, statusColor, state, width, height)
		case ViewDashboard:
			drawDashboard(books, state, width, height)
		case ViewDuplicates:
			duplicates := getDuplicateList(books)
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
			drawDuplicates(books, duplicates, selectedIndex, scrollOffset, subSelectedIndex, activePanel, statusMsg, statusColor, width, height)

		case ViewReader:
			drawReader(activeReaderBook, readerLines, readerScrollIndex, readerSearchQuery, isReaderSearching, readerMatches, readerMatchIndex, state, width, height)
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
					state.ReadingTime[activeReaderBook.SHA1] += duration
					state.BookProgress[activeReaderBook.SHA1] = readerScrollIndex
					saveState(statePath, state)
					
					currentView = ViewExplorer
					selectedIndex = 0
					scrollOffset = 0
					filteredBooks = filterBooks(books, state, searchQuery)
					statusMsg = "Closed Reader. Logged " + formatDuration(duration) + " reading session."
					statusColor = "\033[32m"
				}
			} else if isSearching {
				isSearching = false
				searchQuery = ""
				filteredBooks = filterBooks(books, state, searchQuery)
				selectedIndex = 0
				scrollOffset = 0
				statusMsg = "Search cleared."
				statusColor = "\033[36m"
			} else if currentView != ViewExplorer {
				currentView = ViewExplorer
				activePanel = PanelLeft
				selectedIndex = 0
				scrollOffset = 0
				subSelectedIndex = 0
				filteredBooks = filterBooks(books, state, searchQuery)
				statusMsg = "Returned to Main Explorer."
				statusColor = "\033[32m"
			}
			continue
		}

		// Quit app global
		if keys[0] == 'q' && !isSearching && currentView != ViewReader {
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
					readerMatches = searchReaderText(readerLines, readerSearchQuery)
				}
			} else if keys[0] >= 32 && keys[0] <= 126 { // Typing
				readerSearchQuery += string(keys)
				readerMatches = searchReaderText(readerLines, readerSearchQuery)
			}
			continue
		}

		// --- 3. INPUT HANDLING FOR GLOBAL EXPLORER SEARCH MODE ---
		if isSearching && currentView == ViewExplorer {
			if keys[0] == 13 || keys[0] == 10 { // Enter locks search
				isSearching = false
				statusMsg = "Search locked. Tap 'Esc' to clear search query."
				statusColor = "\033[32m"
			} else if keys[0] == 127 || keys[0] == 8 { // Backspace
				if len(searchQuery) > 0 {
					searchQuery = searchQuery[:len(searchQuery)-1]
					filteredBooks = filterBooks(books, state, searchQuery)
					selectedIndex = 0
					scrollOffset = 0
				}
			} else if keys[0] >= 32 && keys[0] <= 126 { // Normal typing
				searchQuery += string(keys)
				filteredBooks = filterBooks(books, state, searchQuery)
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
					if state.Notes[activeReaderBook.SHA1] != "" {
						fmt.Printf("Current note: \033[1;33m%s\033[0m\n\n", state.Notes[activeReaderBook.SHA1])
					}
					
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("✍️  Enter Note: ")
					noteText, _ := reader.ReadString('\n')
					noteText = strings.TrimSpace(noteText)
					
					if noteText != "" {
						state.Notes[activeReaderBook.SHA1] = noteText
						saveState(statePath, state)
					}

					newRawState, err := term.MakeRaw(fd)
					if err == nil {
						oldState = newRawState
					}
				case 'q': // Exit reader
					duration := int(time.Since(readerStartTime).Seconds())
					state.ReadingTime[activeReaderBook.SHA1] += duration
					state.BookProgress[activeReaderBook.SHA1] = readerScrollIndex
					saveState(statePath, state)
					
					currentView = ViewExplorer
					selectedIndex = 0
					scrollOffset = 0
					filteredBooks = filterBooks(books, state, searchQuery)
					statusMsg = "Closed Reader. Logged " + formatDuration(duration) + " reading session."
					statusColor = "\033[32m"
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
					statusColor = "\033[32m"
					drawExplorer(filteredBooks, selectedIndex, scrollOffset, isSearching, searchQuery, statusMsg, statusColor, state, width, height)
					
					if state.Statuses[b.SHA1] == "" || state.Statuses[b.SHA1] == "Inbox" {
						state.Statuses[b.SHA1] = "Reading"
						saveState(statePath, state)
					}

					err := openBookInSumatra(b)
					if err != nil {
						statusMsg = "Sumatra Launch failed: " + err.Error()
						statusColor = "\033[31m"
					} else {
						statusMsg = "Opened in SumatraPDF. Book status transitioned to 'Reading'."
						statusColor = "\033[32m"
						filteredBooks = filterBooks(books, state, searchQuery)
					}
				}
			}

			// Letter commands
			if action == "" {
				switch keys[0] {
				case 's': // Search
					isSearching = true
					searchQuery = ""
					statusMsg = "Search active. Type keywords, press 'Esc' to exit, 'Enter' to lock."
					statusColor = "\033[33m"
				case 'v': // Launch TUI Reader & transition status
					if len(filteredBooks) > 0 {
						b := filteredBooks[selectedIndex]
						statusMsg = "Extracting book text natively: " + b.FileName + "..."
						statusColor = "\033[33m"
						drawExplorer(filteredBooks, selectedIndex, scrollOffset, isSearching, searchQuery, statusMsg, statusColor, state, width, height)

						rawText, err := extractBookRawText(b)
						if err != nil {
							statusMsg = "TUI Reader failed: " + err.Error() + " (Use SumatraPDF instead)"
							statusColor = "\033[31m"
						} else {
							if state.Statuses[b.SHA1] == "" || state.Statuses[b.SHA1] == "Inbox" {
								state.Statuses[b.SHA1] = "Reading"
								saveState(statePath, state)
							}

							activeReaderBook = b
							activeReaderRawText = rawText
							lastWrapWidth = 0 // Force re-wrap calculation immediately
							readerStartTime = time.Now()
							
							readerScrollIndex = state.BookProgress[b.SHA1]

							currentView = ViewReader
						}
					}
				case 'd': // Dashboard
					currentView = ViewDashboard
				case 'u': // Duplicates
					currentView = ViewDuplicates
					selectedIndex = 0
					scrollOffset = 0
					subSelectedIndex = 0
					activePanel = PanelLeft
					statusMsg = "Duplicates Deck loaded."
					statusColor = "\033[35m"
				case 'i': // Ingest directory scan
					_ = term.Restore(fd, oldState)
					fmt.Print("\033[H\033[2J") // Clear
					fmt.Println("📖 \033[1;36mAURA PORTABLE INGESTION ENGINE\033[0m")
					fmt.Println("----------------------------------------------------------------------")
					fmt.Println("Enter the absolute directory path to scan for ebooks/documents")
					fmt.Println("(or press Enter to scan the current working directory):")
					fmt.Println()
					
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("📂 Path: ")
					scanDir, _ := reader.ReadString('\n')
					scanDir = strings.TrimSpace(scanDir)
					
					if scanDir == "" {
						scanDir, _ = os.Getwd()
					}
					
					fmt.Printf("\n🔍 Scanning directory recursively: %s...\n", scanDir)
					newBooks, err := scanDirectoryForBooks(scanDir)
					if err != nil {
						fmt.Printf("\033[31m[ERROR] Scan failed: %v\033[0m\n", err)
						fmt.Println("\nPress Enter to return to TUI...")
						_, _ = reader.ReadString('\n')
					} else {
						err = appendBooksToCatalog(csvPath, newBooks)
						if err != nil {
							fmt.Printf("\033[31m[ERROR] Ingestion failed: %v\033[0m\n", err)
						} else {
							fmt.Printf("\n\033[32m✔ [SUCCESS] Successfully scanned & parsed %d books!\033[0m\n", len(newBooks))
							books, _ = loadBooks(csvPath)
						}
						fmt.Println("\nPress Enter to reload TUI...")
						_, _ = reader.ReadString('\n')
					}

					newRawState, err := term.MakeRaw(fd)
					if err == nil {
						oldState = newRawState
					}
					filteredBooks = filterBooks(books, state, searchQuery)
					selectedIndex = 0
					scrollOffset = 0
				case 'w': // Toggle status
					if len(filteredBooks) > 0 {
						b := filteredBooks[selectedIndex]
						currStatus := state.Statuses[b.SHA1]
						if currStatus == "" {
							currStatus = "Inbox"
						}

						var nextStatus string
						switch currStatus {
						case "Inbox":
							wipCount := 0
							for _, s := range state.Statuses {
								if s == "Reading" {
									wipCount++
								}
							}
							if wipCount >= 2 {
								statusMsg = "⛔ WIP Limit Exceeded! Archive or finish one of your active reading books first!"
								statusColor = "\033[31m"
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

						state.Statuses[b.SHA1] = nextStatus
						saveState(statePath, state)
						filteredBooks = filterBooks(books, state, searchQuery)
						statusMsg = "Status updated: " + b.Title + " -> " + nextStatus
						statusColor = "\033[32m"
					}
				}
			}
		}


		// --- 7. DUPLICATES VIEW DECK LOGIC ---
		if currentView == ViewDuplicates {
			duplicates := getDuplicateList(books)
			
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
					groupBooks := getBooksByGroup(books, activeGrp)
					if subSelectedIndex < len(groupBooks)-1 {
						subSelectedIndex++
					}
				}
			case "RIGHT", "TAB":
				if activePanel == PanelLeft && len(duplicates) > 0 {
					activePanel = PanelRight
					subSelectedIndex = 0
					statusMsg = "Focused Right Panel."
					statusColor = "\033[32m"
				}
			case "LEFT":
				if activePanel == PanelRight {
					activePanel = PanelLeft
					statusMsg = "Focused Left Panel."
					statusColor = "\033[35m"
				}
			case "ENTER":
				if activePanel == PanelRight && len(duplicates) > 0 {
					activeGrp := duplicates[selectedIndex]
					groupBooks := getBooksByGroup(books, activeGrp)
					if subSelectedIndex < len(groupBooks) {
						target := groupBooks[subSelectedIndex]
						statusMsg = "Opening to visually compare in SumatraPDF: " + target.FileName + "..."
						statusColor = "\033[32m"
						drawDuplicates(books, duplicates, selectedIndex, scrollOffset, subSelectedIndex, activePanel, statusMsg, statusColor, width, height)
						
						_ = openBookInSumatra(target)
					}
				}
			}

			// Pruning and Panel switching actions
			if activePanel == PanelRight {
				if keys[0] == 'd' && len(duplicates) > 0 {
					activeGrp := duplicates[selectedIndex]
					groupBooks := getBooksByGroup(books, activeGrp)
					if len(groupBooks) > 1 && subSelectedIndex < len(groupBooks) {
						target := groupBooks[subSelectedIndex]
						
						statusMsg = "Deleting duplicate copy..."
						statusColor = "\033[33m"
						drawDuplicates(books, duplicates, selectedIndex, scrollOffset, subSelectedIndex, activePanel, statusMsg, statusColor, width, height)

						err := deleteDuplicateSecurely(target)
						if err != nil {
							statusMsg = "Delete failed: " + err.Error()
							statusColor = "\033[31m"
						} else {
							books = removeBookFromMem(books, target.FullPath)
							statusMsg = "🧹 [DELETED] Duplicate pruned successfully!"
							statusColor = "\033[32m"
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
					statusColor = "\033[32m"
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
					statusColor = "\033[35m"
				case 'd':
					currentView = ViewExplorer
					selectedIndex = 0
					scrollOffset = 0
					statusMsg = "Returned to main Explorer."
					statusColor = "\033[32m"
				}
			}
		}
	}
}

func drawExplorer(books []Book, sel, offset int, isSearching bool, query, statusMsg, statusColor string, state *LibraryState, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	drawLine(width, "=", "\033[1;35m")
	fmt.Println("\033[1;36m                             AURA COMPARED TUI ENGINE (GO)                                 \033[0m")
	drawLine(width, "=", "\033[1;35m")
	
	// WIP Active Focus Deck
	wipSlots := []string{}
	for _, b := range books {
		if state.Statuses[b.SHA1] == "Reading" {
			wipSlots = append(wipSlots, b.Title)
		}
	}
	fmt.Print("⚡ \033[1;32m[WIP FOCUS DECK (Max 2)]\033[0m ")
	if len(wipSlots) == 0 {
		fmt.Println("\033[37mEmpty. (Highlight a book and press 'w' to set Reading Focus!)\033[0m")
	} else {
		for i, w := range wipSlots {
			if len(w) > 30 {
				w = w[:27] + "..."
			}
			fmt.Printf("\033[1;33mSlot %d: %s\033[0m", i+1, w)
			if i < len(wipSlots)-1 {
				fmt.Print("  |  ")
			}
		}
		fmt.Println()
	}
	drawLine(width, "-", "\033[38;5;244m")

	titleWidth := width - 48
	if titleWidth < 20 {
		titleWidth = 20
	}
	
	headerFormat := fmt.Sprintf("\033[1;37m%%-3s | %%-%ds | %%-18s | %%-9s | %%-6s\033[0m\n", titleWidth)
	fmt.Printf(headerFormat, "Idx", "Book Title", "Author", "Status", "Format")
	drawLine(width, "-", "\033[38;5;244m")

	visibleRows := height - 14
	if visibleRows < 5 {
		visibleRows = 5
	}

	if len(books) == 0 {
		fmt.Println()
		fmt.Println("    \033[1;36mWelcome to Aura TUI! Your Native Portable Library is Ready.\033[0m")
		fmt.Println()
		fmt.Println("    To get started, you can automatically index your ebook files:")
		fmt.Println("    1. Press \033[1;32m'i'\033[0m to enter the directory auto-discovery scanner.")
		fmt.Println("    2. Provide the absolute directory path of your book collection.")
		fmt.Println("    3. Aura will recursively crawl, clean, fingerprint, and load them instantly!")
		fmt.Println()
		fmt.Println("    Alternatively, you can drop a catalog file named \033[1;33m'books_catalog.csv'\033[0m")
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
				statusColorCode = "\033[1;33m"
			case "Read":
				statusColorCode = "\033[1;32m"
			case "Reference":
				statusColorCode = "\033[1;35m"
			}

			if idx == sel {
				rowFormat := fmt.Sprintf("\033[7;36m%%03d | %%-%ds | %%-18s | %%-9s | %%-6s\033[0m\n", titleWidth)
				fmt.Printf(rowFormat, idx+1, title, author, bStatus, b.Format)
			} else {
				rowFormat := fmt.Sprintf("\033[36m%%03d\033[0m | %%-%ds | %%-18s | %%s%%-9s\033[0m | %%-6s\n", titleWidth)
				fmt.Printf(rowFormat, idx+1, title, author, statusColorCode, bStatus, b.Format)
			}
		}
	}

	drawLine(width, "-", "\033[38;5;244m")

	// Search bar rendering
	if isSearching {
		fmt.Printf("🔍 \033[1;33mSEARCH FILTER:\033[0m %s▮\n", query)
	} else if query != "" {
		fmt.Printf("🔍 \033[1;36mACTIVE FILTER:\033[0m %s (Press 'Esc' to clear or change)\n", query)
	} else {
		fmt.Println("🔍 Type query directly in search mode by pressing 's'")
	}

	drawLine(width, "=", "\033[38;5;244m")
	
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

func drawDashboard(books []Book, state *LibraryState, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	drawLine(width, "=", "\033[1;35m")
	fmt.Println("\033[1;36m                          AURA LIBRARY DYNAMIC METRICS                             \033[0m")
	drawLine(width, "=", "\033[1;35m")

	totalMB := 0.0
	formats := make(map[string]int)
	categories := make(map[string]int)
	statuses := make(map[string]int)
	gibberishCount := 0
	duplicateGroups := make(map[string]bool)

	for _, b := range books {
		totalMB += b.SizeMB
		formats[b.Format]++
		categories[b.LocationCategory]++
		
		bStatus := state.Statuses[b.SHA1]
		if bStatus == "" {
			bStatus = "Inbox"
		}
		statuses[bStatus]++

		if b.FilenameGibberish == "Y" {
			gibberishCount++
		}
		if b.DuplicateGroup != "" {
			duplicateGroups[b.DuplicateGroup] = true
		}
	}

	// Calculate reading time
	totalReadingSeconds := 0
	for _, t := range state.ReadingTime {
		totalReadingSeconds += t
	}
	hours := totalReadingSeconds / 3600
	minutes := (totalReadingSeconds % 3600) / 60

	fmt.Printf("📊 \033[1;32mTOTAL ASSETS UNDER MANAGEMENT:\033[0m %d Files | %.2f GB\n", len(books), totalMB/1024.0)
	fmt.Printf("⏱️  \033[1;36mBEHAVIORAL INVESTMENT TIME:\033[0m %d Hours, %d Minutes Spent Engaging Natively!\n", hours, minutes)
	fmt.Printf("📁 \033[1;33mCOGNITIVE LOAD OUTSTANDING:\033[0m %d Duplicate Groups | %d Gibberish Filenames\n\n", len(duplicateGroups), gibberishCount)

	fmt.Println("\033[1;34m--- BEHAVIORAL PROGRESS BARS ---\033[0m")
	printStatusBar("Inbox (Hoarded / Unread)", statuses["Inbox"], len(books), "\033[31m", width)
	printStatusBar("Reading Now (Active Focus)", statuses["Reading"], len(books), "\033[33m", width)
	printStatusBar("Read (Fully Digested)", statuses["Read"], len(books), "\033[32m", width)
	printStatusBar("Reference Shelf (Parked)", statuses["Reference"], len(books), "\033[35m", width)
	fmt.Println()

	fmt.Println("\033[1;37m--- Category File Distribution ---\033[0m")
	catRow := 0
	for cat, count := range categories {
		if cat == "" {
			cat = "Unsorted / General"
		}
		fmt.Printf("  \033[36m%-25s\033[0m: %3d books   ", cat, count)
		catRow++
		if catRow%2 == 0 {
			fmt.Println()
		}
	}
	if catRow%2 != 0 {
		fmt.Println()
	}

	drawLine(width, "=", "\033[1;35m")
	
	// v5 HUD for Dashboard
	fmt.Println("🔔 LOG: Loaded library dashboard statistics.")
	fmt.Println("\033[1;35m💡 GUIDE: [Esc] Back to Explorer | [u] Duplicates | [q] Exit App\033[0m")
}

// Interactive Tag and Category selector view
func drawTags(tagsAndCats []string, sel, offset int, statusMsg, statusColor string, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	drawLine(width, "=", "\033[1;35m")
	fmt.Println("\033[1;36m                             AURA SUBJECT & TAG BROWSER                                    \033[0m")
	drawLine(width, "=", "\033[1;35m")

	visibleTags := height - 8
	if visibleTags < 5 {
		visibleTags = 5
	}

	if len(tagsAndCats) == 0 {
		fmt.Println("\n                    \033[31mNo unique categories or tags surfaced in catalog.\033[0m\n")
	} else {
		fmt.Println("Scroll using arrow keys (or j/k) and press 'Enter' to filter main explorer by this tag:")
		drawLine(width, "-", "\033[38;5;244m")
		
		end := offset + visibleTags
		if end > len(tagsAndCats) {
			end = len(tagsAndCats)
		}

		for idx := offset; idx < end; idx++ {
			t := tagsAndCats[idx]
			if idx == sel {
				fmt.Printf("\033[7;35m%03d | 🏷️  %s\033[0m\n", idx+1, t)
			} else {
				fmt.Printf("\033[35m%03d\033[0m | 🏷️  %s\n", idx+1, t)
			}
		}
	}

	drawLine(width, "=", "\033[38;5;244m")
	
	// v5 HUD for Tag Browser
	fmt.Printf("%s🔔 LOG: %s\033[0m\n", statusColor, statusMsg)
	fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll Tags | [Enter] Filter Library | [Esc] Back to Explorer | [d] Dashboard | [u] Duplicates | [r] Renamer | [q] Exit App\033[0m")
}

// Side-by-Side Dual Panel Duplicate Comparison View
func drawDuplicates(books []Book, duplicates []string, sel, offset, subSel int, activePanel FocusPanel, statusMsg, statusColor string, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	drawLine(width, "=", "\033[1;35m")
	fmt.Println("\033[1;31m                          AURA HARDENED DEDUPLICATION ENGINE                               \033[0m")
	drawLine(width, "=", "\033[1;35m")

	visibleDups := height - 14
	if visibleDups < 5 {
		visibleDups = 5
	}

	if len(duplicates) == 0 {
		fmt.Println("\n                    \033[1;32m✔ [EXCELLENT] Zero duplicate files found on disk!\033[0m\n")
	} else {
		fmt.Println("LEFT: Duplicate Groups list  |  RIGHT: Specific duplicate files (Press Tab to toggle)")
		drawLine(width, "-", "\033[38;5;244m")

		end := offset + visibleDups
		if end > len(duplicates) {
			end = len(duplicates)
		}

		for idx := offset; idx < end; idx++ {
			grp := duplicates[idx]
			groupBooks := getBooksByGroup(books, grp)
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
				fmt.Printf("\033[7;31mGroup %02d %s [DUP-ID: %-5s] %-40s | %d copies\033[0m\n", idx+1, panelIndicator, grp, title, len(groupBooks))
			} else {
				fmt.Printf("\033[31mGroup %02d\033[0m   [DUP-ID: %-5s] %-40s | %d copies\n", idx+1, grp, title, len(groupBooks))
			}
		}

		drawLine(width, "-", "\033[38;5;244m")

		if sel < len(duplicates) {
			activeGrp := duplicates[sel]
			groupBooks := getBooksByGroup(books, activeGrp)
			
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
					fmt.Printf("\033[7;32m%s\033[0m\n", rowContent)
				} else {
					fmt.Printf("%s\n", rowContent)
				}
			}
		}
	}

	drawLine(width, "=", "\033[38;5;244m")
	
	// v5 HUD for Duplicates (Dynamic key guide based on panel focus!)
	fmt.Printf("%s🔔 LOG: %s\033[0m\n", statusColor, statusMsg)
	if activePanel == PanelLeft {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll Groups | [Tab/→/l] Focus Copies | [Esc] Back to Explorer | [d] Dashboard | [q] Exit App\033[0m")
	} else {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll Copies | [Enter] SumatraPDF | [d] PRUNE (Delete) Copy | [←/h] Focus Groups | [Esc] Back to Explorer\033[0m")
	}
}

func drawGibberish(gibberish []Book, sel, offset int, statusMsg, statusColor string, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	drawLine(width, "=", "\033[1;35m")
	fmt.Println("\033[1;33m                            AURA METADATA GIBBERISH RENAMER                                \033[0m")
	drawLine(width, "=", "\033[1;35m")

	visibleGibb := height - 12
	if visibleGibb < 5 {
		visibleGibb = 5
	}

	if len(gibberish) == 0 {
		fmt.Println("\n                    \033[1;32m✔ [EXCELLENT] Zero gibberish filenames outstanding!\033[0m\n")
	} else {
		fmt.Printf("Surfaced \033[1;33m%d Gibberish Filenames\033[0m. Scroll, select, and press 'y' to safely rename:\n\n", len(gibberish))

		rawColWidth := int(float64(width) * 0.35)
		if rawColWidth < 20 {
			rawColWidth = 20
		}
		titleColWidth := width - rawColWidth - 12
		if titleColWidth < 20 {
			titleColWidth = 20
		}

		headerFormat := fmt.Sprintf("\033[1;37m%%-3s | Raw: %%-%ds | Meta Title: %%-%ds\033[0m\n", rawColWidth, titleColWidth)
		fmt.Printf(headerFormat, "Idx", "Filename", "Parsed Title")
		drawLine(width, "-", "\033[38;5;244m")

		end := offset + visibleGibb
		if end > len(gibberish) {
			end = len(gibberish)
		}

		for idx := offset; idx < end; idx++ {
			g := gibberish[idx]
			rawName := g.FileName
			if len(rawName) > rawColWidth {
				rawName = rawName[:rawColWidth-3] + "..."
			}
			metaTitle := g.Title
			if len(metaTitle) > titleColWidth {
				metaTitle = metaTitle[:titleColWidth-3] + "..."
			}

			if idx == sel {
				rowFormat := fmt.Sprintf("\033[7;33m%%03d | Raw: %%-%ds | Meta Title: %%-%ds\033[0m\n", rawColWidth, titleColWidth)
				fmt.Printf(rowFormat, idx+1, rawName, metaTitle)
			} else {
				rowFormat := fmt.Sprintf("\033[33m%%03d\033[0m | Raw: %%-%ds | Meta Title: %%-%ds\n", idx+1, rawName, metaTitle)
				fmt.Printf(rowFormat, idx+1, rawName, metaTitle)
			}
		}

		drawLine(width, "-", "\033[38;5;244m")
		if sel < len(gibberish) {
			target := gibberish[sel]
			proposedName := sanitizeFilename(target.Title) + "." + target.Format
			fmt.Println("\033[1;36mProposed Safe Disk Action:\033[0m")
			fmt.Printf("  Source File : \033[31m%s\033[0m\n", target.FileName)
			fmt.Printf("  Destination : \033[32m%s\033[0m\n", proposedName)
			fmt.Printf("  Directory   : \033[38;5;244m%s\033[0m\n", target.FullPath)
		}
	}

	drawLine(width, "=", "\033[38;5;244m")
	
	// v5 HUD for Renamer
	fmt.Printf("%s🔔 LOG: %s\033[0m\n", statusColor, statusMsg)
	fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll Files | [y] Safe Rename on Disk | [Esc] Back to Explorer | [d] Dashboard | [t] Tags | [u] Duplicates | [q] Exit App\033[0m")
}

// Draw Console-Native Ebook Reader (ViewReader with Highlighting, Notes, and Find panel)
// Draw Console-Native Ebook Reader (ViewReader with Highlighting, Notes, and Find panel)
func drawReader(b Book, lines []string, scrollIdx int, findQuery string, isSearching bool, matches []int, matchIdx int, state *LibraryState, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear screen
	drawLine(width, "=", "\033[1;35m")
	
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
	
	fmt.Printf("\033[1;36m📖 %-50s\033[0m | \033[1;33mProgress: %3.1f%%\033[0m (Line %d/%d)\n", titleDisplay, percentage, scrollIdx+1, len(lines))
	drawLine(width, "=", "\033[1;35m")

	if len(lines) == 0 {
		fmt.Println("\n                    \033[31m[EMPTY] No text could be extracted from this book.\033[0m\n")
	} else {
		end := scrollIdx + visibleLines
		if end > len(lines) {
			end = len(lines)
		}

		for idx := scrollIdx; idx < end; idx++ {
			lineText := lines[idx]
			
			// Highlight search queries in glowing yellow in the TUI reader!
			if findQuery != "" {
				re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(findQuery))
				if err == nil {
					lineText = re.ReplaceAllString(lineText, "\033[7;33m$0\033[0m\033[37m")
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
				fmt.Printf(" \033[1;32m%04d\033[0m | \033[37m%s\033[0m\n", idx+1, lineText)
			} else {
				fmt.Printf(" \033[38;5;244m%04d\033[0m | %s\n", idx+1, lineText)
			}
		}
	}

	drawLine(width, "=", "\033[1;35m")
	
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
	fmt.Printf("\033[1;33m%s\033[0m\n", noteText)
	drawLine(width, "-", "\033[38;5;244m")

	// Find panel rendering
	if findQuery != "" {
		matchStr := "0 matches"
		if len(matches) > 0 {
			matchStr = fmt.Sprintf("%d/%d matches", matchIdx+1, len(matches))
		}
		fmt.Printf("🔍 \033[1;33mFIND IN BOOK:\033[0m %s▮ (%s) | Press 'n'/'N' to cycle\n", findQuery, matchStr)
	} else {
		fmt.Println("🔍 Press '/' to Find text in this book | Press 'e' to Edit Note")
	}

	drawLine(width, "=", "\033[1;35m")
	// v5 HUD for Reader
	if isSearching {
		fmt.Println("\033[1;35m💡 GUIDE: [Type] to find | [Backspace] Delete char | [Enter] Go to First Match | [Esc] Cancel search\033[0m")
	} else {
		fmt.Println("\033[1;35m💡 GUIDE: [↑/↓/j/k] Scroll | [Space/d] Page Down | [u/b] Page Up | [/] Find Text | [n/N] Next Match | [e] Edit Note | [Esc/q] Exit Reader\033[0m")
	}
}

// Search text helper for TUI Reader
func searchReaderText(lines []string, query string) []int {
	var indices []int
	if query == "" {
		return indices
	}
	q := strings.ToLower(query)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), q) {
			indices = append(indices, i)
		}
	}
	return indices
}

func formatDuration(sec int) string {
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	min := sec / 60
	s := sec % 60
	return fmt.Sprintf("%dm %ds", min, s)
}

// 📦 NATIVE IN-MEMORY EPUB TEXT EXTRACTOR - RAW PARAGRAPHS EXTRACTOR
func extractBookRawText(b Book) (string, error) {
	cleanPath := filepath.Clean(b.FullPath)
	ext := strings.ToLower(filepath.Ext(cleanPath))

	var rawText string

	if ext == ".txt" {
		content, err := os.ReadFile(cleanPath)
		if err != nil { return "", err }
		rawText = string(content)
	} else if ext == ".epub" {
		r, err := zip.OpenReader(cleanPath)
		if err != nil { return "", err }
		defer r.Close()

		var textBuilder strings.Builder
		for _, f := range r.File {
			fExt := strings.ToLower(filepath.Ext(f.Name))
			if fExt == ".xhtml" || fExt == ".html" || fExt == ".htm" {
				rc, err := f.Open()
				if err != nil { continue }
				
				content, err := io.ReadAll(rc)
				rc.Close()
				if err != nil { continue }

				cleanHTML := stripHTMLTags(string(content))
				textBuilder.WriteString(cleanHTML)
				textBuilder.WriteString("\n\n")
			}
		}
		rawText = textBuilder.String()
	} else {
		return "", fmt.Errorf("format %s not supported inside console (Use SumatraPDF instead)", b.Format)
	}
	return rawText, nil
}

// wrapText wraps paragraphs of raw text to a specified column width
func wrapText(rawText string, wrapWidth int) []string {
	var wrappedLines []string
	scanner := bufioScanner(rawText)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		words := strings.Fields(trimmed)
		currentLine := ""
		for _, w := range words {
			if len(currentLine)+len(w)+1 > wrapWidth {
				wrappedLines = append(wrappedLines, currentLine)
				currentLine = w
			} else {
				if currentLine == "" {
					currentLine = w
				} else {
					currentLine += " " + w
				}
			}
		}
		if currentLine != "" {
			wrappedLines = append(wrappedLines, currentLine)
		}
	}
	return wrappedLines
}

func bufioScanner(s string) *bufioMockScanner {
	return &bufioMockScanner{lines: strings.Split(s, "\n"), idx: -1}
}

type bufioMockScanner struct {
	lines []string
	idx   int
}

func (s *bufioMockScanner) Scan() bool {
	s.idx++
	return s.idx < len(s.lines)
}

func (s *bufioMockScanner) Text() string {
	return s.lines[s.idx]
}

func stripHTMLTags(s string) string {
	var builder strings.Builder
	inTag := false
	inEntity := false
	var entityBuilder strings.Builder

	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
			builder.WriteRune(' ') 
		} else if !inTag {
			if r == '&' {
				inEntity = true
				entityBuilder.Reset()
			} else if r == ';' && inEntity {
				inEntity = false
				switch entityBuilder.String() {
				case "nbsp": builder.WriteRune(' ')
				case "lt": builder.WriteRune('<')
				case "gt": builder.WriteRune('>')
				case "amp": builder.WriteRune('&')
				case "quot": builder.WriteRune('"')
				case "apos": builder.WriteRune('\'')
				}
			} else if inEntity {
				entityBuilder.WriteRune(r)
			} else {
				builder.WriteRune(r)
			}
		}
	}
	return builder.String()
}

func getTagsAndCategories(books []Book) []string {
	tracker := make(map[string]int)
	
	for _, b := range books {
		if b.LocationCategory != "" {
			catKey := b.LocationCategory
			tracker[catKey]++
		}
		
		if b.Tags != "" {
			tags := strings.Split(b.Tags, ",")
			for _, t := range tags {
				trimmed := strings.TrimSpace(t)
				if trimmed != "" {
					tracker[trimmed]++
				}
			}
		}
	}

	var sorted []string
	for k, count := range tracker {
		sorted = append(sorted, fmt.Sprintf("%s (%d)", k, count))
	}
	
	sort.Strings(sorted)
	return sorted
}

func findSumatraPath() string {
	// 1. Check user custom APP path
	customPath := `C:\Users\Laxmi Share Market\AppData\Local\SumatraPDF\SumatraPDF.exe`
	if _, err := os.Stat(customPath); err == nil {
		return customPath
	}
	// 2. Check localized AppData
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		p := filepath.Join(localAppData, "SumatraPDF", "SumatraPDF.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 3. Check Program Files
	pFiles := os.Getenv("ProgramFiles")
	if pFiles != "" {
		p := filepath.Join(pFiles, "SumatraPDF", "SumatraPDF.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 4. Fallback system path
	if path, err := exec.LookPath("SumatraPDF.exe"); err == nil {
		return path
	}
	return ""
}

// ⚡ SYSTEM NATIVE FILE EXECUTION (Secure, formatted, and strictly SumatraPDF-driven!)
func openBookInSumatra(b Book) error {
	cleanPath := filepath.Clean(b.FullPath)
	
	// Bounds check
	if !strings.HasPrefix(cleanPath, `D:\`) && !strings.HasPrefix(cleanPath, `C:\`) {
		return fmt.Errorf("security bounds: path out-of-bounds")
	}

	// Verify existence
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found on disk")
	}

	s := findSumatraPath()
	if s != "" {
		cmd := exec.Command(s, cleanPath)
		return cmd.Start()
	}
	
	// Fallback to standard OS default viewer
	cmd := exec.Command("cmd", "/c", "start", "", cleanPath)
	return cmd.Start()
}

// 🧹 SECURE DEDUPLICATION EXECUTOR
func deleteDuplicateSecurely(b Book) error {
	cleanPath := filepath.Clean(b.FullPath)

	// Bound boundaries
	if !strings.HasPrefix(cleanPath, `D:\backups\`) && !strings.HasPrefix(cleanPath, `D:\C shifted\`) && !strings.HasPrefix(cleanPath, `D:\Library\`) {
		return fmt.Errorf("security bounds: deletions only allowed in backups, C shifted, or Library to protect core files")
	}

	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist")
	}

	return os.Remove(cleanPath)
}

// 🔄 SECURE FILENAME RENAMER
func renameFileSecurely(b Book, newName string) error {
	cleanSrc := filepath.Clean(b.FullPath)
	dir := filepath.Dir(cleanSrc)
	cleanDst := filepath.Clean(filepath.Join(dir, newName))

	if !strings.HasPrefix(cleanSrc, `D:\`) && !strings.HasPrefix(cleanSrc, `C:\`) {
		return fmt.Errorf("security exception: destination boundaries breached")
	}

	if _, err := os.Stat(cleanSrc); os.IsNotExist(err) {
		return fmt.Errorf("source file not found")
	}

	if _, err := os.Stat(cleanDst); !os.IsNotExist(err) {
		return fmt.Errorf("destination file name already exists")
	}

	return os.Rename(cleanSrc, cleanDst)
}

// Helpers
func getDuplicateList(books []Book) []string {
	dupMap := make(map[string]bool)
	var list []string
	for _, b := range books {
		if b.DuplicateGroup != "" && !dupMap[b.DuplicateGroup] {
			dupMap[b.DuplicateGroup] = true
			list = append(list, b.DuplicateGroup)
		}
	}
	return list
}

func getBooksByGroup(books []Book, grp string) []Book {
	var groupBooks []Book
	for _, b := range books {
		if b.DuplicateGroup == grp {
			groupBooks = append(groupBooks, b)
		}
	}
	return groupBooks
}

func getGibberishList(books []Book) []Book {
	var list []Book
	for _, b := range books {
		if b.FilenameGibberish == "Y" {
			list = append(list, b)
		}
	}
	return list
}

func removeBookFromMem(books []Book, fullPath string) []Book {
	var list []Book
	for _, b := range books {
		if b.FullPath != fullPath {
			list = append(list, b)
		}
	}
	return list
}

func updateBookNameInMem(books []Book, fullPath, newName string) []Book {
	for i, b := range books {
		if b.FullPath == fullPath {
			dir := filepath.Dir(fullPath)
			books[i].FileName = newName
			books[i].FullPath = filepath.Join(dir, newName)
			books[i].FilenameGibberish = "N"
			break
		}
	}
	return books
}

func sanitizeFilename(s string) string {
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	res := s
	for _, c := range invalidChars {
		res = strings.ReplaceAll(res, c, "")
	}
	res = strings.TrimSpace(res)
	if len(res) > 80 {
		res = res[:77] + "..."
	}
	if res == "" {
		res = "Untitled_Document"
	}
	return res
}

// printStatusBar draws the progress bars in the metrics dashboard
func printStatusBar(label string, count, total int, color string, termWidth int) {
	percentage := 0.0
	if total > 0 {
		percentage = (float64(count) / float64(total)) * 100.0
	}
	barTotalWidth := termWidth - 45
	if barTotalWidth < 10 {
		barTotalWidth = 10
	}
	if barTotalWidth > 50 {
		barTotalWidth = 50 // clamp to max 50 for aesthetic sanity
	}
	barWidth := int((percentage / 100.0) * float64(barTotalWidth))
	bar := ""
	for i := 0; i < barTotalWidth; i++ {
		if i < barWidth {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	fmt.Printf("  %-25s : %s[%s] %3d (%3.1f%%)\033[0m\n", label, color, bar, count, percentage)
}

// drawLine draws a terminal-wide horizontal line of arbitrary characters
func drawLine(width int, char string, colorCode string) {
	if width <= 0 {
		return
	}
	fmt.Printf("%s%s\033[0m\n", colorCode, strings.Repeat(char, width))
}

// readInput reads raw terminal input bytes and sends them to a channel
func readInput(out chan []byte) {
	var buf [10]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return
		}
		out <- buf[:n]
	}
}

// filterBooks filters books based on title, author, category, status, and tags
func filterBooks(books []Book, state *LibraryState, query string) []Book {
	var res []Book
	q := strings.ToLower(query)
	for _, b := range books {
		bStatus := state.Statuses[b.SHA1]
		if bStatus == "" {
			bStatus = "Inbox"
		}
		
		matchesQuery := q == "" || 
			strings.Contains(strings.ToLower(b.Title), q) || 
			strings.Contains(strings.ToLower(b.Author), q) || 
			strings.Contains(strings.ToLower(b.LocationCategory), q) || 
			strings.Contains(strings.ToLower(b.Tags), q) ||
			strings.Contains(strings.ToLower(bStatus), q)

		if matchesQuery {
			res = append(res, b)
		}
	}
	return res
}

// computeQuickSHA1 hashes the first 32KB of a file to generate a quick fingerprint
func computeQuickSHA1(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	h := sha1.New()
	buffer := make([]byte, 32*1024)
	n, _ := file.Read(buffer)
	h.Write(buffer[:n])
	return fmt.Sprintf("%x", h.Sum(nil))
}

// scanDirectoryForBooks recursively crawls a directory for ebook formats
func scanDirectoryForBooks(dir string) ([]Book, error) {
	var books []Book
	
	cleanDir := filepath.Clean(dir)
	if _, err := os.Stat(cleanDir); err != nil {
		return nil, err
	}

	hashTracker := make(map[string][]int)

	err := filepath.Walk(cleanDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".pdf" || ext == ".epub" || ext == ".txt" {
			sizeMB := float64(info.Size()) / (1024.0 * 1024.0)
			
			shaVal := computeQuickSHA1(path)
			if shaVal == "" {
				return nil
			}

			fileName := info.Name()
			format := strings.TrimPrefix(ext, ".")
			
			title := strings.TrimSuffix(fileName, ext)
			author := "Unknown"
			
			if strings.Contains(title, " - ") {
				parts := strings.SplitN(title, " - ", 2)
				title = strings.TrimSpace(parts[0])
				author = strings.TrimSpace(parts[1])
			}

			category := filepath.Base(filepath.Dir(path))
			if category == "." || category == filepath.Base(cleanDir) {
				category = "Library"
			}

			gibberish := "N"
			if matched, _ := regexp.MatchString(`^[a-fA-F0-9\-]{25,}`, title); matched || len(title) > 20 && !strings.Contains(title, " ") {
				gibberish = "Y"
			}

			books = append(books, Book{
				Title:             title,
				Author:            author,
				Format:            format,
				SizeMB:            sizeMB,
				LocationCategory:  category,
				DuplicateGroup:    "",
				FilenameGibberish: gibberish,
				SHA1:              shaVal,
				FileName:          fileName,
				FullPath:          path,
				Modified:          info.ModTime().Format("2006-01-02"),
			})
			
			hashTracker[shaVal] = append(hashTracker[shaVal], len(books)-1)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	groupCounter := 1
	for _, indices := range hashTracker {
		if len(indices) > 1 {
			groupTag := fmt.Sprintf("G%02d", groupCounter)
			for _, idx := range indices {
				books[idx].DuplicateGroup = groupTag
			}
			groupCounter++
		}
	}

	return books, nil
}

// appendBooksToCatalog appends scanned unique files to the local CSV database
func appendBooksToCatalog(path string, newBooks []Book) error {
	existingMap := make(map[string]bool)
	var allBooks []Book
	
	file, err := os.Open(path)
	if err == nil {
		file.Close()
		loaded, err := loadBooks(path)
		if err == nil {
			allBooks = loaded
			for _, b := range loaded {
				existingMap[b.SHA1] = true
			}
		}
	}

	addedCount := 0
	for _, nb := range newBooks {
		if !existingMap[nb.SHA1] {
			allBooks = append(allBooks, nb)
			existingMap[nb.SHA1] = true
			addedCount++
		}
	}

	if addedCount == 0 && len(allBooks) > 0 {
		return nil
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	header := []string{"Title", "Author", "Format", "SizeMB", "LocationCategory", "DuplicateGroup", "FilenameGibberish", "SHA1", "FileName", "FullPath", "Modified"}
	_ = writer.Write(header)

	for _, b := range allBooks {
		record := []string{
			b.Title,
			b.Author,
			b.Format,
			fmt.Sprintf("%.2f", b.SizeMB),
			b.LocationCategory,
			b.DuplicateGroup,
			b.FilenameGibberish,
			b.SHA1,
			b.FileName,
			b.FullPath,
			b.Modified,
		}
		_ = writer.Write(record)
	}
	return nil
}
