package qizmo

type EndpointState struct {
	playerNames        [clientDynamicTokenCount]string
	s2cResyncRequested bool
}

func NewEndpointState() *EndpointState {
	return &EndpointState{}
}

func (s *EndpointState) observePlayerNames(names [clientDynamicTokenCount]string) {
	s.playerNames = names
}

func (s *EndpointState) requestS2CResync() {
	s.s2cResyncRequested = true
}

func (s *EndpointState) needsS2CResync() bool {
	return s.s2cResyncRequested
}

func (s *EndpointState) clearS2CResyncRequest() {
	s.s2cResyncRequested = false
}

type sequenceTracker struct {
	last    uint32
	started bool
}

func (s *sequenceTracker) observe(sequence uint32) {
	if !s.started {
		s.last = sequence
		s.started = true
		return
	}
	forward := (sequence - s.last) & sequenceMask
	if forward != 0 && forward < sequenceHalfRange {
		s.last = sequence
	}
}
