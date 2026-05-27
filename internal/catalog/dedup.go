package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GetDuplicateList(books []Book) []string {
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

func GetBooksByGroup(books []Book, grp string) []Book {
	var groupBooks []Book
	for _, b := range books {
		if b.DuplicateGroup == grp {
			groupBooks = append(groupBooks, b)
		}
	}
	return groupBooks
}

func RemoveBookFromMem(books []Book, fullPath string) []Book {
	var list []Book
	for _, b := range books {
		if b.FullPath != fullPath {
			list = append(list, b)
		}
	}
	return list
}

func DeleteDuplicateSecurely(b Book) error {
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
