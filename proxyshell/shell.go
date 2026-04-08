package proxyshell

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/svc"
)

type State int

const (
	StateShell State = iota
	StateConnecting
	StateLive
)

type Shell struct {
	log      *log.Logger
	mu       sync.Mutex
	sessions map[string]*session

	newHandlers      []func(string)
	stringHandlers   []func(string, string)
	upstreamHandlers []func(string, packet.Packet)
	errorHandlers    []func(string, error)
}

type identity struct {
	name          string
	team          string
	topColor      byte
	bottomColor   byte
	clientVersion string
	spectator     bool
	userInfo      *infostring.InfoString
	connect       *connect.Command
}

type session struct {
	state    State
	target   string
	remote   *upstreamSession
	remoteID uint64
	client   identity

	reconnectBy time.Time
}

var errNoUpstream = errors.New("no upstream session")

func New(logger *log.Logger) *Shell {
	return &Shell{
		log:      logger,
		sessions: map[string]*session{},
	}
}

func (s *Shell) IsInShell(clientID string) bool {
	return s.state(clientID) == StateShell
}

func (s *Shell) OnNew(h func(string)) {
	s.newHandlers = append(s.newHandlers, h)
}

func (s *Shell) OnString(h func(string, string)) {
	s.stringHandlers = append(s.stringHandlers, h)
}

func (s *Shell) OnUpstream(h func(string, packet.Packet)) {
	s.upstreamHandlers = append(s.upstreamHandlers, h)
}

func (s *Shell) OnUpstreamError(h func(string, error)) {
	s.errorHandlers = append(s.errorHandlers, h)
}

func (s *Shell) Connect(clientID, target string) bool {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	sameTarget := ss.target == target
	alreadyConnecting := ss.state == StateConnecting && ss.remote != nil
	name := ss.client.name

	if sameTarget && alreadyConnecting {
		s.mu.Unlock()
		return false
	}

	ss.target = target
	ss.state = StateConnecting
	ss.reconnectBy = time.Now().Add(5 * time.Second)

	s.mu.Unlock()

	s.logf("%s (%s) connecting to %s", name, clientID, target)
	s.shutdownRemote(clientID)

	return true
}

func (s *Shell) Disconnect(clientID string) {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	ss.target = ""
	ss.state = StateShell
	ss.reconnectBy = time.Time{}
	name := ss.client.name

	s.mu.Unlock()

	s.logf("%s (%s) disconnected from upstream", name, clientID)
	s.shutdownRemote(clientID)
}

func (s *Shell) Shutdown() {
	s.mu.Lock()

	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}

	s.mu.Unlock()

	for _, id := range ids {
		s.shutdownRemote(id)
	}
}

func (s *Shell) WriteUpstream(
	clientID string,
	pkt packet.Packet,
) error {
	s.mu.Lock()
	remote := s.sessionLocked(clientID).remote
	s.mu.Unlock()

	if remote == nil {
		return errNoUpstream
	}

	return remote.WritePacket(pkt)
}

func (s *Shell) AttachUpstream(clientID string) bool {
	s.ensureRemoteConnected(clientID)

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sessionLocked(clientID).target != ""
}

func (s *Shell) resetIfStale(clientID string) bool {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	reconnect := !ss.reconnectBy.IsZero() && time.Now().Before(ss.reconnectBy)
	if reconnect || ss.state == StateShell {
		ss.reconnectBy = time.Time{}
		s.mu.Unlock()
		return false
	}

	old := ss.remote
	name := ss.client.name
	ss.state = StateShell
	ss.target = ""
	ss.remote = nil
	ss.reconnectBy = time.Time{}

	s.mu.Unlock()

	if old != nil {
		old.Close()
	}

	s.logf("%s (%s) closed stale upstream session", name, clientID)
	return true
}

func (s *Shell) logf(format string, args ...any) {
	if s.log != nil {
		s.log.Printf(format, args...)
	}
}

func (s *Shell) sessionLocked(clientID string) *session {
	ss, ok := s.sessions[clientID]
	if ok {
		return ss
	}

	ss = &session{
		state: StateShell,
	}

	s.sessions[clientID] = ss

	return ss
}

func (s *Shell) state(clientID string) State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sessionLocked(clientID).state
}

func (s *Shell) ensureRemoteConnected(clientID string) {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	target := ss.target
	remote := ss.remote

	s.mu.Unlock()

	if target == "" || remote != nil {
		return
	}

	go s.connectRemote(clientID, target)
}

func (s *Shell) connectRemote(clientID, target string) {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	ss.remoteID++
	sessionID := ss.remoteID
	old := ss.remote
	ss.remote = nil
	client := ss.client
	ss.state = StateConnecting

	s.mu.Unlock()

	if old != nil {
		old.Close()
	}

	cmd := cloneConnectCommand(client.connect)
	if cmd == nil {
		cmd = buildUpstreamConnect(client)
	}

	rc, err := newUpstreamSession(s.log, cmd)
	if err != nil {
		for _, h := range s.errorHandlers {
			h(clientID, err)
		}
		return
	}

	s.mu.Lock()

	ss = s.sessionLocked(clientID)
	if sessionID != ss.remoteID {
		s.mu.Unlock()
		rc.Close()
		return
	}

	ss.remote = rc

	s.mu.Unlock()

	cb := func(pkt packet.Packet) {
		s.mu.Lock()

		ss := s.sessionLocked(clientID)
		if ss.remote != rc || sessionID != ss.remoteID {
			s.mu.Unlock()
			return
		}

		if _, ok := pkt.(*svc.GameData); ok {
			wasLive := ss.state == StateLive
			ss.state = StateLive
			name := ss.client.name

			s.mu.Unlock()

			if !wasLive {
				s.logf("%s (%s) connected to %s", name, clientID, target)
			}

			for _, h := range s.upstreamHandlers {
				h(clientID, pkt)
			}

			return
		}

		s.mu.Unlock()

		for _, h := range s.upstreamHandlers {
			h(clientID, pkt)
		}
	}

	if err := rc.Connect(target, cb); err != nil {
		for _, h := range s.errorHandlers {
			h(clientID, err)
		}
	}
}

func (s *Shell) shutdownRemote(clientID string) {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	old := ss.remote
	ss.remote = nil
	ss.remoteID++
	target := ss.target
	name := ss.client.name

	if target == "" {
		ss.state = StateShell
	}

	s.mu.Unlock()

	if old != nil {
		s.logf("%s (%s) closed upstream session", name, clientID)
		old.Close()
	}
}
