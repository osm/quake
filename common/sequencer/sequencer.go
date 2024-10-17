package sequencer

import (
	"errors"
	"time"

	"github.com/osm/quake/packet/command"
)

var ErrRateLimit = errors.New("rate limit reached")

type State byte

const (
	Disconnected State = 0
	Handshake    State = 1
	Connecting   State = 2
	Connected    State = 3
)

type Sequencer struct {
	ping      int16
	lastWrite time.Time

	state State

	incomingSeq     uint32
	incomingAck     uint32
	lastReliableSeq uint32
	outgoingSeq     uint32

	isIncomingAckReliable bool
	isOutgoingSeqReliable bool
	isIncomingSeqReliable bool

	outgoingCommands    []command.Command
	outgoingCommandsBuf []command.Command
}

func New(opts ...Option) *Sequencer {
	s := Sequencer{
		ping: 999,
	}

	for _, opt := range opts {
		opt(&s)
	}

	return &s
}

func (s *Sequencer) SetState(state State) { s.state = state }
func (s *Sequencer) GetState() State      { return s.state }
func (s *Sequencer) SetPing(ping int16)   { s.ping = ping }
func (s *Sequencer) GetPing() int16       { return s.ping }

func (s *Sequencer) Reset() {
	s.incomingSeq = 0
	s.incomingAck = 0
	s.lastReliableSeq = 0
	s.outgoingSeq = 0

	s.isIncomingAckReliable = false
	s.isOutgoingSeqReliable = false
	s.isIncomingSeqReliable = false

	s.outgoingCommands = []command.Command{}
	s.outgoingCommandsBuf = []command.Command{}
}

func (s *Sequencer) Process(
	incomingSeq, incomingAck uint32,
	cmds []command.Command,
) (uint32, uint32, []command.Command, error) {
	s.incoming(incomingSeq, incomingAck)
	return s.outgoing(cmds)
}

func (s *Sequencer) incoming(incomingSeq, incomingAck uint32) {
	isIncomingSeqReliable := incomingSeq>>31 == 1
	isIncomingAckReliable := incomingAck>>31 == 1

	incomingSeq = incomingSeq & 0x7fffffff
	incomingAck = incomingAck & 0x7fffffff

	if incomingSeq < s.incomingSeq {
		return
	}

	if isIncomingAckReliable == s.isOutgoingSeqReliable {
		s.outgoingCommandsBuf = []command.Command{}
	}

	if isIncomingSeqReliable {
		s.isIncomingSeqReliable = !s.isIncomingSeqReliable
	}

	s.incomingSeq = incomingSeq
	s.incomingAck = incomingAck
	s.isIncomingAckReliable = isIncomingAckReliable
}

func (s *Sequencer) outgoing(cmds []command.Command) (uint32, uint32, []command.Command, error) {
	s.outgoingCommands = append(s.outgoingCommands, cmds...)

	if s.state == Connected && time.Since(s.lastWrite).Milliseconds() < int64(s.ping) {
		return 0, 0, nil, ErrRateLimit
	}

	var isReliable bool

	if s.incomingAck > s.lastReliableSeq &&
		s.isIncomingAckReliable != s.isOutgoingSeqReliable {
		isReliable = true
	}

	if len(s.outgoingCommandsBuf) == 0 && len(s.outgoingCommands) > 0 {
		s.outgoingCommandsBuf = s.outgoingCommands
		s.isOutgoingSeqReliable = !s.isOutgoingSeqReliable
		isReliable = true
		s.outgoingCommands = []command.Command{}
	}

	outgoingSeq := s.outgoingSeq
	if isReliable {
		outgoingSeq = s.outgoingSeq | (1 << 31)
	}

	outgoingAck := s.incomingSeq
	if s.isIncomingSeqReliable {
		outgoingAck = s.incomingSeq | (1 << 31)
	}

	outgoingCmds := []command.Command{}

	s.outgoingSeq++

	if isReliable {
		outgoingCmds = append(outgoingCmds, s.outgoingCommandsBuf...)
		s.lastReliableSeq = s.outgoingSeq
	}

	s.lastWrite = time.Now()
	return outgoingSeq, outgoingAck, outgoingCmds, nil
}
