package ui

import (
	"fmt"
	"strings"

	"github.com/osm/quake/packet/command/centerprint"
)

func (s *Session) Render() string {
	if !s.visible {
		return ""
	}
	if s.status != "" {
		return s.renderNotification()
	}

	if s.prompt != nil {
		return s.renderPrompt()
	}

	f := s.normalize()
	if f == nil {
		return ""
	}

	width, rows := s.dimensions()
	inside := width - 2
	style := s.Style.normalized()
	lines := style.header(f.page.Title, inside)
	if f.page.confirm {
		lines = append(lines, renderButtons(f.items, f.selected, inside, style),
			border(style.BottomLeft, style.Bottom, style.BottomRight, inside))
		return strings.Join(lines, "\n")
	}

	end := min(f.top+rows, len(f.items))
	for i := f.top; i < end; i++ {
		for gap := 1; i > f.top && gap < style.LineHeight; gap++ {
			lines = append(lines, style.row(strings.Repeat(" ", inside)))
		}
		lines = append(lines, s.renderItem(f.items[i], i == f.selected, inside, style))
	}
	if len(f.items) == 0 {
		lines = append(lines, style.textRow("  (empty)", inside))
	}

	lines = append(lines, border(style.BottomLeft, style.Bottom, style.BottomRight, inside))
	if len(f.items) > rows {
		lines = append(lines, style.TextColor.text(fit(fmt.Sprintf("%d-%d / %d", f.top+1, end, len(f.items)), width)))
	}

	return strings.Join(lines, "\n")
}

func (s *Session) renderItem(item Item, selected bool, width int, style Style) string {
	if item.Separator {
		return border(style.SeparatorLeft, style.Separator, style.SeparatorRight, width)
	}
	color := style.TextColor
	if item.action() && !item.disabled(s) {
		color = style.ActionColor
	}
	marker := style.Unselected
	if item.action() {
		marker = style.Action
	}
	if selected {
		marker = style.Selected
	}

	available := width - 2
	value := ""
	if item.progress != nil {
		value = renderProgress(item.progress(s), available/2, style)
	} else if item.Value != nil {
		value = clean(item.Value(s))
	} else if item.Submenu != nil {
		value = string([]byte{style.Submenu})
	}

	label := color.text(fit(item.Label, available))
	if value != "" {
		value = value[:min(len(value), available/2)]
		label = color.text(fit(item.Label, available-len(value)-1)) + " " + style.TextColor.text(value)
	}

	if selected {
		label = style.SelectedColor.text(label)
	}
	return style.row(string([]byte{marker, ' '}) + label)
}

func (s *Session) CenterPrint() *centerprint.Command {
	return &centerprint.Command{String: s.Render()}
}

func clean(text string) string {
	b := []byte(text)
	for i, c := range b {
		if c < 32 || c == 127 || c == 255 {
			b[i] = ' '
		}
	}

	return string(b)
}

func fit(text string, width int) string {
	text = clean(text)
	if len(text) > width {
		return text[:width]
	}

	return text + strings.Repeat(" ", width-len(text))
}

func center(text string, width int) string {
	text = clean(text)
	text = text[:min(len(text), width)]
	return fit(strings.Repeat(" ", (width-len(text))/2)+text, width)
}
