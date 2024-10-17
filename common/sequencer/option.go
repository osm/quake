package sequencer

type Option func(*Sequencer)

func WithIncomingSeq(incomingSeq uint32) Option {
	return func(s *Sequencer) {
		s.incomingSeq = incomingSeq
	}
}

func WithOutgoingSeq(outgoingSeq uint32) Option {
	return func(s *Sequencer) {
		s.outgoingSeq = outgoingSeq
	}
}

func WithPing(ping int16) Option {
	return func(s *Sequencer) {
		s.ping = ping
	}
}
