package ui

// Stable Item.Key values keep selection on the same item across refreshes.
// Keep the callback read-only; session calls can re-enter it.
func List(title string, items func(*Session) []Item) *Page {
	return &Page{Title: title, source: items}
}
