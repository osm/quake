package ui

import "strings"

type Style struct {
	LineHeight     int
	ActionColor    Color
	SelectedColor  Color
	TextColor      Color
	TopLeft        byte
	Top            byte
	TopRight       byte
	Left           byte
	Right          byte
	SeparatorLeft  byte
	Separator      byte
	SeparatorRight byte
	BottomLeft     byte
	Bottom         byte
	BottomRight    byte
	Selected       byte
	Action         byte
	Unselected     byte
	Submenu        byte
	ProgressLeft   byte
	ProgressThumb  byte
	ProgressTrack  byte
	ProgressRight  byte
}

func DefaultStyle() Style {
	return Style{
		LineHeight:     1,
		ActionColor:    ColorWhite,
		SelectedColor:  ColorBrown,
		TextColor:      ColorOriginal,
		TopLeft:        0x1d,
		Top:            0x1e,
		TopRight:       0x1f,
		Left:           ' ',
		Right:          ' ',
		SeparatorLeft:  ' ',
		Separator:      ' ',
		SeparatorRight: ' ',
		BottomLeft:     0x1d,
		Bottom:         0x1e,
		BottomRight:    0x1f,
		Selected:       0x8d,
		Action:         ' ',
		Unselected:     ' ',
		Submenu:        0x8d,
		ProgressLeft:   0x80,
		ProgressThumb:  0x83,
		ProgressTrack:  0x81,
		ProgressRight:  0x82,
	}
}

func (s Style) normalized() Style {
	d := DefaultStyle()
	return Style{
		LineHeight:     s.lineHeight(),
		ActionColor:    s.ActionColor.normalized(d.ActionColor),
		SelectedColor:  s.SelectedColor.normalized(d.SelectedColor),
		TextColor:      s.TextColor.normalized(d.TextColor),
		TopLeft:        styleGlyph(s.TopLeft, d.TopLeft),
		Top:            styleGlyph(s.Top, d.Top),
		TopRight:       styleGlyph(s.TopRight, d.TopRight),
		Left:           styleGlyph(s.Left, d.Left),
		Right:          styleGlyph(s.Right, d.Right),
		SeparatorLeft:  styleGlyph(s.SeparatorLeft, d.SeparatorLeft),
		Separator:      styleGlyph(s.Separator, d.Separator),
		SeparatorRight: styleGlyph(s.SeparatorRight, d.SeparatorRight),
		BottomLeft:     styleGlyph(s.BottomLeft, d.BottomLeft),
		Bottom:         styleGlyph(s.Bottom, d.Bottom),
		BottomRight:    styleGlyph(s.BottomRight, d.BottomRight),
		Selected:       styleGlyph(s.Selected, d.Selected),
		Action:         styleGlyph(s.Action, d.Action),
		Unselected:     styleGlyph(s.Unselected, d.Unselected),
		Submenu:        styleGlyph(s.Submenu, d.Submenu),
		ProgressLeft:   styleGlyph(s.ProgressLeft, d.ProgressLeft),
		ProgressThumb:  styleGlyph(s.ProgressThumb, d.ProgressThumb),
		ProgressTrack:  styleGlyph(s.ProgressTrack, d.ProgressTrack),
		ProgressRight:  styleGlyph(s.ProgressRight, d.ProgressRight),
	}
}

func (s Style) lineHeight() int {
	return max(1, min(s.LineHeight, 3))
}

func styleGlyph(glyph, fallback byte) byte {
	switch glyph {
	case 0, '\n', '\r', 0xff:
		return fallback
	default:
		return glyph
	}
}

func border(left, fill, right byte, width int) string {
	return string([]byte{left}) + strings.Repeat(string([]byte{fill}), width) + string([]byte{right})
}

func (s Style) row(text string) string {
	return string([]byte{s.Left}) + text + string([]byte{s.Right})
}

func (s Style) header(title string, width int) []string {
	return []string{
		border(s.TopLeft, s.Top, s.TopRight, width),
		s.row(s.ActionColor.text(center(title, width))),
		border(s.SeparatorLeft, s.Separator, s.SeparatorRight, width),
	}
}

func (s Style) textRow(text string, width int) string {
	return s.row(s.TextColor.text(fit(text, width)))
}
