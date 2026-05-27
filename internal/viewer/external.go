package viewer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func OpenBookInSumatra(b catalog.Book) error {
	cleanPath := filepath.Clean(b.FullPath)
	
	// Bounds check
	if !strings.HasPrefix(cleanPath, `D:\`) && !strings.HasPrefix(cleanPath, `C:\`) {
		return fmt.Errorf("security bounds: path out-of-bounds")
	}

	// Verify existence
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found on disk")
	}

	s := FindSumatraPath()
	if s != "" {
		cmd := exec.Command(s, cleanPath)
		return cmd.Start()
	}
	
	// Fallback to standard OS default viewer
	cmd := exec.Command("cmd", "/c", "start", "", cleanPath)
	return cmd.Start()
}
