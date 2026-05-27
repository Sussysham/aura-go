package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/term"

	"aura-go/internal/catalog"
	"aura-go/internal/config"
	"aura-go/internal/state"
	"aura-go/internal/tui"
)

func main() {
	configPath := "aura.toml"
	if exePath, err := os.Executable(); err == nil {
		configPath = filepath.Join(filepath.Dir(exePath), "aura.toml")
	}
	_ = config.LoadConfig(configPath)

	var initialMsg = "Ready."
	var initialColor = config.ActiveTheme.SuccessSt

	stateData, err := state.LoadState(config.StatePath)
	if err != nil {
		initialMsg = "⚠️ " + err.Error()
		initialColor = config.ActiveTheme.ErrorSt
	}

	books, err := catalog.LoadBooks(config.CSVPath)
	if err != nil {
		books = []catalog.Book{}
		initialMsg = "⚠️ Catalog load failed: " + err.Error()
		initialColor = config.ActiveTheme.ErrorSt
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

	tui.RunTUI(fd, books, stateData, oldState, initialMsg, initialColor)
}
