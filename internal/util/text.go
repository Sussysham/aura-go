package util

import (
	"strings"
)

func WrapText(rawText string, wrapWidth int) []string {
	var wrappedLines []string
	scanner := BufioScanner(rawText)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		words := strings.Fields(trimmed)
		currentLine := ""
		for _, w := range words {
			if len(currentLine)+len(w)+1 > wrapWidth {
				wrappedLines = append(wrappedLines, currentLine)
				currentLine = w
			} else {
				if currentLine == "" {
					currentLine = w
				} else {
					currentLine += " " + w
				}
			}
		}
		if currentLine != "" {
			wrappedLines = append(wrappedLines, currentLine)
		}
	}
	return wrappedLines
}

func BufioScanner(s string) *BufioMockScanner {
	return &BufioMockScanner{lines: strings.Split(s, "\n"), idx: -1}
}

type BufioMockScanner struct {
	lines []string
	idx   int
}

func (s *BufioMockScanner) Scan() bool {
	s.idx++
	return s.idx < len(s.lines)
}

func (s *BufioMockScanner) Text() string {
	return s.lines[s.idx]
}
