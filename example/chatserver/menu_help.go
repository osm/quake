package main

import (
	"strings"

	"github.com/osm/quake/ui"
)

func menuHelp(title string, lines ...string) *ui.Page {
	page := ui.TextPage(title, strings.Join(lines, "\n"))
	page.Items = menuFooter()
	return page
}
