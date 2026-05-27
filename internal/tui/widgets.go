package tui

import (
	"fmt"
	"strings"

	"aura-go/internal/catalog"
	"aura-go/internal/state"
)

func DrawLine(width int, char string, colorCode string) {
	if width <= 0 {
		return
	}
	fmt.Printf("%s%s\033[0m\n", colorCode, strings.Repeat(char, width))
}

func PrintStatusBar(label string, count, total int, color string, termWidth int) {
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

func SearchReaderText(lines []string, query string) []int {
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

func FilterBooks(books []catalog.Book, state *state.LibraryState, query string) []catalog.Book {
	var res []catalog.Book
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
