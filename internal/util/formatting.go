package util

import (
	"fmt"
	"strings"
)

func FormatDuration(sec int) string {
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	min := sec / 60
	s := sec % 60
	return fmt.Sprintf("%dm %ds", min, s)
}

func SanitizeFilename(s string) string {
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
