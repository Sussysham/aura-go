package reader

import (
	"html"
	"strings"
)

func StripHTMLTags(s string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
			builder.WriteRune(' ') 
		} else if !inTag {
			builder.WriteRune(r)
		}
	}
	return html.UnescapeString(builder.String())
}
