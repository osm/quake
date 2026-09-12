package ui

import (
	"fmt"
	"strings"

	"github.com/osm/quake/protocol"
)

func (c *Connection) promptInput(text string, seq uint32, fresh, capture bool) (bool, error) {
	// Late copies of submitted text must not become ordinary game chat.
	seq &= protocol.QWSequenceMask
	if !capture && (!c.textReceived || newer(seq, c.textIncoming)) {
		return false, nil
	}

	value, matched, err := promptText(text)
	if !matched || !fresh || !c.ready {
		return matched, nil
	}

	c.textReceived, c.textIncoming = true, seq
	if c.session.status != "" {
		return true, nil
	}
	if err != nil {
		if c.session.prompt != nil {
			c.session.prompt.reject(err)
		}
		return true, err
	}
	if c.session.prompt == nil {
		return true, nil
	}

	return true, c.session.Submit(value)
}

func promptText(text string) (string, bool, error) {
	text = strings.TrimSpace(text)
	if len(text) < 3 || !strings.EqualFold(text[:3], "say") {
		return "", false, nil
	}
	if len(text) > 3 && text[3] != ' ' && text[3] != '\t' {
		return "", false, nil
	}

	value := strings.TrimSpace(text[3:])
	if !strings.HasPrefix(value, "\"") {
		return value, true, nil
	}
	if len(value) < 2 || !strings.HasSuffix(value, "\"") {
		return "", true, fmt.Errorf("Unclosed quote")
	}

	return value[1 : len(value)-1], true, nil
}
