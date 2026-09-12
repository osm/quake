package ui

import (
	"fmt"
	"strings"
)

// Notify opens a notification dialog. Select or Back dismisses it.
func (s *Session) Notify(message string) {
	s.status = message
	s.statusTop = 0
	if message != "" && s.root != nil {
		s.visible = true
	}
}

func (s *Session) notificationLayout() (width, rows int, lines []string) {
	width, rows = s.dimensions()
	// Leave room for a blank line and the OK button within the content budget.
	rows = min(rows, 10)
	lines = wrapText(s.status, width-2)
	s.statusTop = max(0, min(s.statusTop, len(lines)-rows))
	return width, rows, lines
}

func (s *Session) renderNotification() string {
	width, rows, message := s.notificationLayout()
	inside := width - 2
	style := s.Style.normalized()
	lines := style.header("Notification", inside)
	end := min(s.statusTop+rows, len(message))
	for _, line := range message[s.statusTop:end] {
		lines = append(lines, style.textRow(center(line, inside), inside))
	}
	lines = append(lines, style.textRow("", inside),
		renderButtons([]Item{{Label: "OK"}}, 0, inside, style),
		border(style.BottomLeft, style.Bottom, style.BottomRight, inside))
	if len(message) > rows {
		lines = append(lines, style.TextColor.text(fit(fmt.Sprintf("%d-%d / %d", s.statusTop+1, end, len(message)), width)))
	}
	return strings.Join(lines, "\n")
}

func (s *Session) handleNotification(event Event) {
	_, rows, lines := s.notificationLayout()
	switch event {
	case Select, Back:
		s.status, s.statusTop = "", 0
	case Up:
		s.statusTop--
	case Down:
		s.statusTop++
	case PageUp:
		s.statusTop -= rows
	case PageDown:
		s.statusTop += rows
	case Home:
		s.statusTop = 0
	case End:
		s.statusTop = len(lines) - rows
	}
	s.statusTop = max(0, min(s.statusTop, len(lines)-rows))
}
