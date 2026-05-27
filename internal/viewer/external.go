package viewer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"aura-go/internal/catalog"
)

func FindSumatraPath() string {
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

func OpenBookCrossPlatform(b catalog.Book) error {
	cleanPath := filepath.Clean(b.FullPath)
	
	// Security bounds check (operating system sensitive)
	if runtime.GOOS == "windows" {
		cleanedLower := strings.ToLower(cleanPath)
		if !strings.HasPrefix(cleanedLower, `d:\`) && !strings.HasPrefix(cleanedLower, `c:\`) {
			return fmt.Errorf("security bounds: path out-of-bounds")
		}
	}

	// Verify existence
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found on disk")
	}

	switch runtime.GOOS {
	case "windows":
		s := FindSumatraPath()
		if s != "" {
			cmd := exec.Command(s, cleanPath)
			return cmd.Start()
		}
		cmd := exec.Command("cmd", "/c", "start", "", cleanPath)
		return cmd.Start()

	case "darwin": // macOS standard opener
		cmd := exec.Command("open", cleanPath)
		return cmd.Start()

	case "linux": // Linux standard desktop opener
		cmd := exec.Command("xdg-open", cleanPath)
		return cmd.Start()

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
