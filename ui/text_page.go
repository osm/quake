package ui

import "strings"

// Page.Items are appended after the wrapped text.
func TextPage(title, text string) *Page {
	page := &Page{Title: title, scroll: true}
	page.source = func(s *Session) []Item {
		width, _ := s.dimensions()
		var items []Item
		for _, line := range wrapText(text, width-4) {
			items = append(items, Item{Label: line})
		}
		return append(items, page.Items...)
	}
	return page
}

func wrapText(text string, width int) []string {
	var lines []string
	text = strings.ReplaceAll(text, "\r\n", "\n")
	for _, paragraph := range strings.Split(text, "\n") {
		lines = append(lines, wrapParagraph(paragraph, width)...)
	}
	return lines
}

func wrapParagraph(text string, width int) []string {
	var lines []string
	line := ""
	// Only ASCII spaces separate words: high-bit bytes are Quake glyphs.
	for _, word := range strings.FieldsFunc(clean(text), func(r rune) bool { return r == ' ' }) {
		if line != "" && len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = ""
		}
		for len(word) > width {
			lines = append(lines, word[:width])
			word = word[width:]
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	return append(lines, line)
}
