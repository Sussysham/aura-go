package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"aura-go/internal/util"
)

type LibraryState struct {
	Statuses     map[string]string `json:"statuses"`
	Notes        map[string]string `json:"notes"`
	BookProgress map[string]int    `json:"book_progress"` // Maps Book SHA1 to last scroll line index
	ReadingTime  map[string]int    `json:"reading_time"`   // Maps Book SHA1 to total seconds read
}

func LoadState(path string) (*LibraryState, error) {
	state := &LibraryState{
		Statuses:     make(map[string]string),
		Notes:        make(map[string]string),
		BookProgress: make(map[string]int),
		ReadingTime:  make(map[string]int),
	}
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return state, nil // Fresh startup
	}

	file, err := os.Open(path)
	if err != nil {
		return state, fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(state)
	if err != nil {
		corruptPath := path + ".corrupt"
		_ = file.Close()
		_ = util.CopyFile(path, corruptPath)
		return state, fmt.Errorf("library state corrupted, backed up to %s: %w", filepath.Base(corruptPath), err)
	}
	
	return state, nil
}

func SaveState(path string, state *LibraryState) error {
	if _, err := os.Stat(path); err == nil {
		backupPath := path + ".bak"
		_ = util.CopyFile(path, backupPath)
	}

	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, "library_state_*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempName := tempFile.Name()
	defer func() {
		if tempFile != nil {
			tempFile.Close()
			_ = os.Remove(tempName)
		}
	}()

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(state)
	if err != nil {
		return fmt.Errorf("failed to encode JSON state: %w", err)
	}

	err = tempFile.Close()
	tempFile = nil
	if err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	err = os.Rename(tempName, path)
	if err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}
