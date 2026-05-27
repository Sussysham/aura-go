package config

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General     GeneralConfig     `toml:"general"`
	Paths       PathsConfig       `toml:"paths"`
	Scanner     ScannerConfig     `toml:"scanner"`
	Theme       ThemeConfig       `toml:"theme"`
	Keybindings KeybindingsConfig `toml:"keybindings"`
}

type GeneralConfig struct {
	WIPLimit       int    `toml:"wip_limit"`
	DefaultViewer  string `toml:"default_viewer"`
	ReaderMaxWidth int    `toml:"reader_max_width"`
	ReaderMinWidth int    `toml:"reader_min_width"`
}

type PathsConfig struct {
	Catalog     string `toml:"catalog"`
	State       string `toml:"state"`
	SumatraPath string `toml:"sumatra_path"`
}

type ScannerConfig struct {
	Extensions         []string `toml:"extensions"`
	GibberishThreshold int      `toml:"gibberish_threshold"`
}

type ThemeConfig struct {
	Name   string            `toml:"name"`
	Custom CustomThemeConfig `toml:"custom"`
}

type CustomThemeConfig struct {
	Header     string `toml:"header"`
	Accent     string `toml:"accent"`
	Selection  string `toml:"selection"`
	Success    string `toml:"success"`
	Warning    string `toml:"warning"`
	Error      string `toml:"error"`
	Muted      string `toml:"muted"`
	Violet     string `toml:"violet"`
	Background string `toml:"background"`
}

type KeybindingsConfig struct {
	Search     string `toml:"search"`
	Dashboard  string `toml:"dashboard"`
	Duplicates string `toml:"duplicates"`
	Ingest     string `toml:"ingest"`
	ThemeCycle string `toml:"theme_cycle"`
	Quit       string `toml:"quit"`
}

var (
	AppConfig Config
	CSVPath   string
	StatePath string
)

func setDefaults() {
	AppConfig.General.WIPLimit = 2
	AppConfig.General.DefaultViewer = "sumatra"
	AppConfig.General.ReaderMaxWidth = 120
	AppConfig.General.ReaderMinWidth = 40

	AppConfig.Scanner.Extensions = []string{".pdf", ".epub", ".txt", ".docx", ".mobi", ".md"}
	AppConfig.Scanner.GibberishThreshold = 25

	AppConfig.Theme.Name = "midnight"

	AppConfig.Keybindings.Search = "s"
	AppConfig.Keybindings.Dashboard = "d"
	AppConfig.Keybindings.Duplicates = "u"
	AppConfig.Keybindings.Ingest = "i"
	AppConfig.Keybindings.ThemeCycle = "t"
	AppConfig.Keybindings.Quit = "q"
}

func LoadConfig(path string) error {
	setDefaults()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Write default template to path
		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)
		_ = encoder.Encode(AppConfig)
		_ = os.WriteFile(path, buf.Bytes(), 0644)
	} else {
		_, err = toml.DecodeFile(path, &AppConfig)
		if err != nil {
			return err
		}
	}

	InitPaths()
	LoadTheme(AppConfig.Theme.Name, AppConfig.Theme.Custom)
	return nil
}

func InitPaths() {
	// If custom paths are specified in config, use them; otherwise, auto-detect local folders.
	if AppConfig.Paths.Catalog != "" {
		CSVPath = AppConfig.Paths.Catalog
	} else {
		originalCSV := `D:\Audit and revam\_inventory\books_catalog.csv`
		if _, err := os.Stat(originalCSV); err == nil {
			CSVPath = originalCSV
		} else {
			currDir, err := os.Getwd()
			if err == nil {
				CSVPath = filepath.Join(currDir, "books_catalog.csv")
			} else {
				CSVPath = "books_catalog.csv"
			}
		}
	}

	if AppConfig.Paths.State != "" {
		StatePath = AppConfig.Paths.State
	} else {
		originalState := `D:\Audit and revam\aura-go\library_state.json`
		if _, err := os.Stat(originalState); err == nil {
			StatePath = originalState
		} else {
			currDir, err := os.Getwd()
			if err == nil {
				StatePath = filepath.Join(currDir, "library_state.json")
			} else {
				StatePath = "library_state.json"
			}
		}
	}
}
