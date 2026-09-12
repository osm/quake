package ui

import (
	"fmt"
	"strings"
)

// Invalid menu commands still match so adapters can consume them instead of
// leaking them into chat.
func ParseCommand(text, namespace string) (event Event, matched bool, err error) {
	if namespace == "" || strings.ContainsAny(namespace, " \t\r\n\x00\";") {
		return "", false, fmt.Errorf("ui: invalid command namespace")
	}

	text, malformedQuote := menuCommandText(text)
	if len(text) < len(namespace) || !strings.EqualFold(text[:len(namespace)], namespace) {
		return "", false, nil
	}

	rest := text[len(namespace):]
	if rest != "" && !strings.ContainsRune(" \t\r\n\x00\";", rune(rest[0])) {
		return "", false, nil
	}
	if malformedQuote || strings.ContainsAny(rest, "\r\n\x00\";") {
		return "", true, fmt.Errorf("ui: invalid menu command")
	}

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return Toggle, true, nil
	}
	if len(fields) != 1 {
		return "", true, fmt.Errorf("ui: expected one menu event")
	}

	event = Event(strings.ToLower(fields[0]))
	switch event {
	case Toggle, Open, Close, Up, Down, Left, Right, Select, Delete, Help, Back, Home, End, PageUp, PageDown, BindStandard:
		return event, true, nil
	default:
		return "", true, fmt.Errorf("ui: unknown event %q", event)
	}
}

func menuCommandText(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if len(text) <= 3 || !strings.EqualFold(text[:3], "say") || (text[3] != ' ' && text[3] != '\t') {
		return text, false
	}

	text = strings.TrimSpace(text[4:])
	if !strings.HasPrefix(text, "\"") {
		return text, false
	}

	text = text[1:]
	if !strings.HasSuffix(text, "\"") {
		return text, true
	}

	return text[:len(text)-1], false
}
