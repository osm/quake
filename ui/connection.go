package ui

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/centerprint"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
)

// Callbacks run under the connection lock; re-entering Connection methods deadlocks.
// Access the owned session only through the connection to avoid races.
type Connection struct {
	mu             sync.Mutex
	session        *Session
	options        ConnectionOptions
	ready          bool
	sent           bool
	outgoing       uint32
	received       bool
	incoming       uint32
	lastFrame      string
	lastSend       time.Time
	clearing       bool
	clearSequences []uint32
	textReceived   bool
	textIncoming   uint32
	bindings       *stufftext.Command
	bindSequences  []uint32
	lastBind       time.Time
}

func NewConnection(session *Session, options ConnectionOptions) (*Connection, error) {
	if session == nil {
		return nil, fmt.Errorf("ui: nil session")
	}

	options, err := options.normalized()
	if err != nil {
		return nil, err
	}

	return &Connection{session: session, options: options}, nil
}

func (c *Connection) Input(p *clc.GameData) []error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fresh := c.acknowledge(p)
	capture := c.session.prompt != nil
	var errs []error
	out := p.Commands[:0]
	for _, raw := range p.Commands {
		cmd, ok := raw.(*stringcmd.Command)
		if !ok {
			out = append(out, raw)
			continue
		}

		capture = capture || c.session.prompt != nil
		matched, err := c.menuInput(cmd.String, fresh)
		if !matched {
			matched, err = c.promptInput(cmd.String, p.Seq, fresh, capture)
		}
		if !matched {
			out = append(out, raw)
			continue
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	p.Commands = out
	return errs
}

func (c *Connection) menuInput(text string, fresh bool) (bool, error) {
	event, matched, err := ParseCommand(text, c.options.Namespace)
	if !matched || !fresh {
		return matched, nil
	}
	if err == nil && event == BindStandard {
		return true, c.bindStandard()
	}
	if err == nil && c.ready {
		err = c.session.Handle(event)
		c.keepOpen()
	}

	return true, err
}

// Run last so the packet limit includes all other handlers.
func (c *Connection) Output(p *svc.GameData, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	seq := p.Seq & protocol.QWSequenceMask
	if c.sent && !newer(seq, c.outgoing) {
		return
	}
	c.sent, c.outgoing = true, seq
	c.signOn(p.Commands)
	if !c.ready {
		return
	}

	c.appendBindings(p, now)
	c.keepOpen()
	frame := c.session.Render()
	if frame != "" {
		c.clearing = false
		c.clearSequences = nil
	}
	if frame == "" && c.lastFrame != "" {
		c.clearing = true
	}
	if c.filterCenterPrints(p, frame) {
		return
	}
	if frame == "" && !c.clearing {
		return
	}
	if frame == c.lastFrame && now.Sub(c.lastSend) < c.options.RefreshInterval && !c.promptEditorPending() {
		return
	}

	c.appendFrame(p, frame, now)
}

func (c *Connection) acknowledge(p *clc.GameData) bool {
	seq := p.Seq & protocol.QWSequenceMask
	fresh := !c.received || newer(seq, c.incoming)
	if !fresh {
		return false
	}

	c.received, c.incoming = true, seq
	ack := p.Ack & protocol.QWSequenceMask
	if slices.Contains(c.clearSequences, ack) {
		c.clearing = false
		c.clearSequences = nil
	}
	if slices.Contains(c.bindSequences, ack) {
		c.bindings = nil
		c.bindSequences = nil
	}

	return true
}

func (c *Connection) signOn(commands []command.Command) {
	for _, raw := range commands {
		switch raw.(type) {
		case *serverdata.Command:
			c.session.Reset()
			c.ready = false
			c.lastFrame, c.lastSend = "", time.Time{}
			c.clearing = false
			c.clearSequences = nil
		case *packetentities.Command:
			if c.ready {
				continue
			}
			c.ready = true
			if c.options.OpenOnConnect {
				_ = c.session.Handle(Open)
			}
		}
	}
}

func (c *Connection) filterCenterPrints(p *svc.GameData, frame string) bool {
	serverPrint := false
	out := p.Commands[:0]
	for _, raw := range p.Commands {
		_, isPrint := raw.(*centerprint.Command)
		if isPrint && frame != "" {
			continue
		}
		serverPrint = serverPrint || isPrint
		out = append(out, raw)
	}

	p.Commands = out
	if serverPrint && frame == "" {
		c.lastFrame = ""
		c.clearing = false
		c.clearSequences = nil
		return true
	}

	return false
}

func (c *Connection) appendFrame(p *svc.GameData, frame string, now time.Time) {
	cmd := &centerprint.Command{String: frame}
	size := packetSize(p) + len(cmd.Bytes())
	editor := &stufftext.Command{String: "messagemode\n"}
	if c.promptEditorPending() {
		size += len(editor.Bytes())
	}
	if size > c.options.MaxPacketBytes {
		return
	}

	p.Commands = append(p.Commands, cmd)
	if c.promptEditorPending() {
		p.Commands = append(p.Commands, editor)
		c.session.prompt.edit = false
	}
	c.lastFrame, c.lastSend = frame, now
	if frame != "" {
		return
	}

	if len(c.clearSequences) >= 16 {
		c.clearSequences = c.clearSequences[1:]
	}
	c.clearSequences = append(c.clearSequences, p.Seq&protocol.QWSequenceMask)
}

func newer(a, b uint32) bool {
	d := (a - b) & protocol.QWSequenceMask
	return d != 0 && d < (protocol.QWSequenceMask+1)/2
}

func packetSize(p *svc.GameData) int {
	size := 8
	for _, cmd := range p.Commands {
		size += len(cmd.Bytes())
	}
	return size
}

func (c *Connection) promptEditorPending() bool {
	return c.session.status == "" && c.session.prompt != nil && c.session.prompt.edit
}

func (c *Connection) keepOpen() {
	if c.options.AlwaysVisible {
		_ = c.session.Handle(Open)
	}
}
