package ui

type Page struct {
	Title   string
	Items   []Item
	Help    *Page
	source  func(*Session) []Item
	scroll  bool
	confirm bool
}
