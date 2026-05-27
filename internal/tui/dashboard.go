package tui

import (
	"fmt"

	"aura-go/internal/catalog"
	"aura-go/internal/config"
	"aura-go/internal/state"
)

func DrawDashboard(books []catalog.Book, state *state.LibraryState, width, height int) {
	fmt.Print("\033[H\033[2J") // Clear
	DrawLine(width, "=", config.ColorViolet)
	fmt.Printf("%s                          AURA LIBRARY DYNAMIC METRICS                             \033[0m\n", config.ColorHeader)
	DrawLine(width, "=", config.ColorViolet)

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

	fmt.Printf("📊 %sTOTAL ASSETS UNDER MANAGEMENT:\033[0m %d Files | %.2f GB\n", config.ColorSuccess, len(books), totalMB/1024.0)
	fmt.Printf("⏱️  %sBEHAVIORAL INVESTMENT TIME:\033[0m %d Hours, %d Minutes Spent Engaging Natively!\n", config.ColorHeader, hours, minutes)
	fmt.Printf("📁 %sCOGNITIVE LOAD OUTSTANDING:\033[0m %d Duplicate Groups | %d Gibberish Filenames\n\n", config.ColorWarning, len(duplicateGroups), gibberishCount)

	fmt.Printf("\033[1;34m--- BEHAVIORAL PROGRESS BARS ---\033[0m\n")
	PrintStatusBar("Inbox (Hoarded / Unread)", statuses["Inbox"], len(books), config.ColorErrorSt, width)
	PrintStatusBar("Reading Now (Active Focus)", statuses["Reading"], len(books), config.ColorWarningSt, width)
	PrintStatusBar("Read (Fully Digested)", statuses["Read"], len(books), config.ColorSuccessSt, width)
	PrintStatusBar("Reference Shelf (Parked)", statuses["Reference"], len(books), config.ColorVioletSt, width)
	fmt.Println()

	fmt.Printf("\033[1;37m--- Category File Distribution ---\033[0m\n")
	catRow := 0
	for cat, count := range categories {
		if cat == "" {
			cat = "Unsorted / General"
		}
		fmt.Printf("  %s%-25s\033[0m: %3d books   ", config.ColorHeader, cat, count)
		catRow++
		if catRow%2 == 0 {
			fmt.Println()
		}
	}
	if catRow%2 != 0 {
		fmt.Println()
	}

	DrawLine(width, "=", config.ColorViolet)
	
	// v5 HUD for Dashboard
	fmt.Println("🔔 LOG: Loaded library dashboard statistics.")
	fmt.Println("\033[1;35m💡 GUIDE: [Esc] Back to Explorer | [u] Duplicates | [q] Exit App\033[0m")
}
