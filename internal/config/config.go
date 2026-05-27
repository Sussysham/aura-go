package config

import (
	"os"
	"path/filepath"
)

var (
	CSVPath   string
	StatePath string
)

func InitPaths() {
	originalCSV := `D:\Audit and revam\_inventory\books_catalog.csv`
	originalState := `D:\Audit and revam\aura-go\library_state.json`
	if _, err := os.Stat(originalCSV); err == nil {
		CSVPath = originalCSV
		StatePath = originalState
		return
	}
	
	currDir, err := os.Getwd()
	if err == nil {
		CSVPath = filepath.Join(currDir, "books_catalog.csv")
		StatePath = filepath.Join(currDir, "library_state.json")
	} else {
		CSVPath = "books_catalog.csv"
		StatePath = "library_state.json"
	}
}
