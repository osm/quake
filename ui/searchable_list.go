package ui

import "strings"

// As with List, keep items read-only and use stable keys. Page.Items are not filtered.
func SearchableList(title string, items func(*Session) []Item) *Page {
	page := &Page{Title: title}
	set := func(s *Session, text string) error {
		if s.Current() == page {
			s.stack[len(s.stack)-1].query = strings.TrimSpace(text)
		}
		return nil
	}
	search := TextField("Search", 64, func(s *Session) string {
		return s.stack[len(s.stack)-1].query
	}, set)
	search.Delete = func(s *Session) error { return set(s, "") }
	search.Help = TextPage("Search", "Enter: type a search, then press Enter to apply.\nDelete: clear the search.\nMatches names regardless of case or color.")
	page.source = func(s *Session) []Item {
		query := searchText(s.stack[len(s.stack)-1].query)
		result := []Item{search}
		result = append(result, filterItems(items(s), query)...)
		if len(result) == 1 {
			result = append(result, Item{Label: "(no matches)"})
		}
		return append(result, page.Items...)
	}
	return page
}

func filterItems(items []Item, query string) []Item {
	var result []Item
	for _, item := range items {
		if query != "" && (item.Separator || !strings.Contains(searchText(item.Label), query)) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func searchText(text string) string {
	b := []byte(text)
	for i, ch := range b {
		ch &= 0x7f
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		b[i] = ch
	}
	return string(b)
}
