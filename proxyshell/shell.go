package proxyshell

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/disconnect"
	"github.com/osm/quake/packet/command/s2cconnection"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/command/stufftext"
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

	shellPacketHandlers       []func(string, packet.Packet)
	upstreamPacketHandlers    []func(string, packet.Packet)
	upstreamErrorHandlers     []func(string, error)
	shellInjectHandlers       []func(string, packet.Packet)
	shellResetHandlers        []func(string)
	shellDisconnectedHandlers []func(string)
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

	reconnectBy       time.Time
	pendingNew        bool
	pendingConnect    bool
	pendingDisconnect bool
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

func (s *Shell) OnShellPacket(h func(string, packet.Packet)) {
	s.shellPacketHandlers = append(s.shellPacketHandlers, h)
}

func (s *Shell) OnUpstreamPacket(h func(string, packet.Packet)) {
	s.upstreamPacketHandlers = append(s.upstreamPacketHandlers, h)
}

func (s *Shell) OnUpstreamError(h func(string, error)) {
	s.upstreamErrorHandlers = append(s.upstreamErrorHandlers, h)
}

func (s *Shell) OnShellInjectPacket(h func(string, packet.Packet)) {
	s.shellInjectHandlers = append(s.shellInjectHandlers, h)
}

func (s *Shell) OnShellResetClient(h func(string)) {
	s.shellResetHandlers = append(s.shellResetHandlers, h)
}

func (s *Shell) OnShellDisconnected(h func(string)) {
	s.shellDisconnectedHandlers = append(s.shellDisconnectedHandlers, h)
}

func (s *Shell) ConnectUpstream(clientID, target string) (bool, bool) {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	sameTarget := ss.target == target
	alreadyConnecting := ss.state == StateConnecting
	hasRemote := ss.remote != nil

	if sameTarget && (alreadyConnecting || hasRemote) {
		s.mu.Unlock()
		return false, false
	}

	ss.target = target
	ss.reconnectBy = time.Now().Add(5 * time.Second)
	live := ss.state != StateShell

	s.mu.Unlock()

	if !live {
		s.ensureRemoteConnected(clientID)
		return true, false
	}

	s.emitLocal(clientID, &stufftext.Command{String: "changing\n"})

	s.mu.Lock()
	ss = s.sessionLocked(clientID)
	ss.pendingConnect = true
	ss.pendingDisconnect = false
	s.mu.Unlock()

	return true, true
}

func (s *Shell) DisconnectUpstream(clientID string) bool {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	live := ss.state != StateShell
	if live {
		ss.target = ""
		ss.pendingDisconnect = true
		ss.pendingConnect = false
		s.mu.Unlock()
		s.emitLocal(clientID, &stufftext.Command{String: "changing\n"})
		return true
	}

	ss.target = ""
	ss.state = StateShell
	ss.reconnectBy = time.Time{}
	ss.pendingConnect = false
	ss.pendingDisconnect = false
	name := ss.client.name

	s.mu.Unlock()

	s.logf("%s (%s) disconnected from upstream", name, clientID)
	s.shutdownRemote(clientID)
	return false
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
	ss := s.sessionLocked(clientID)
	remote := ss.remote
	pendingNew := ss.pendingNew
	if pendingNew {
		if game, ok := pkt.(*clc.GameData); ok {
			cmds := make([]command.Command, 0, len(game.Commands)+1)
			cmds = append(cmds, &stringcmd.Command{String: "new"})
			cmds = append(cmds, game.Commands...)
			pkt = &clc.GameData{
				Seq:      game.Seq,
				Ack:      game.Ack,
				QPort:    game.QPort,
				Commands: cmds,
			}
			ss.pendingNew = false
		}
	}
	s.mu.Unlock()

	if remote == nil {
		return errNoUpstream
	}

	return remote.WritePacket(pkt)
}

func (s *Shell) attachUpstream(clientID string) bool {
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

func (s *Shell) emitLocal(clientID string, cmds ...command.Command) {
	if len(cmds) == 0 {
		return
	}

	pkt := &svc.GameData{Commands: cmds}
	for _, h := range s.shellInjectHandlers {
		h(clientID, pkt)
	}
}

func (s *Shell) resetClient(clientID string) {
	for _, h := range s.shellResetHandlers {
		h(clientID)
	}
}

func (s *Shell) emitDisconnected(clientID string) {
	for _, h := range s.shellDisconnectedHandlers {
		h(clientID)
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

	s.logf("%s (%s) connecting to %s", client.name, clientID, target)

	cmd := cloneConnectCommand(client.connect)
	if cmd == nil {
		cmd = buildUpstreamConnect(client)
	}

	rc, err := newUpstreamSession(s.log, cmd)
	if err != nil {
		for _, h := range s.upstreamErrorHandlers {
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

		if oob, ok := pkt.(*svc.Connectionless); ok {
			if _, ok := oob.Command.(*s2cconnection.Command); ok {
				ss.pendingNew = true
			}
		}

		if game, ok := pkt.(*svc.GameData); ok {
			if ss.pendingConnect || ss.pendingDisconnect {
				filtered := make([]command.Command, 0, len(game.Commands))
				for _, raw := range game.Commands {
					if _, ok := raw.(*disconnect.Command); ok {
						continue
					}
					filtered = append(filtered, raw)
				}
				pkt = &svc.GameData{
					Seq:      game.Seq,
					Ack:      game.Ack,
					Commands: filtered,
				}
			}

			wasLive := ss.state == StateLive
			ss.state = StateLive
			name := ss.client.name
			pendingConnect := ss.pendingConnect
			pendingDisconnect := ss.pendingDisconnect

			s.mu.Unlock()

			if !wasLive {
				s.logf("%s (%s) connected to %s", name, clientID, target)
			}

			for _, h := range s.upstreamPacketHandlers {
				h(clientID, pkt)
			}

			if pendingDisconnect {
				s.mu.Lock()
				ss := s.sessionLocked(clientID)
				ss.pendingDisconnect = false
				s.mu.Unlock()

				s.disconnectTransition(clientID)
				s.resetClient(clientID)
				s.emitDisconnected(clientID)
			}

			if pendingConnect {
				s.mu.Lock()
				ss := s.sessionLocked(clientID)
				ss.pendingConnect = false
				s.mu.Unlock()

				s.disconnectTransition(clientID)
				s.resetClient(clientID)
				s.attachUpstream(clientID)
			}

			return
		}

		s.mu.Unlock()

		for _, h := range s.upstreamPacketHandlers {
			h(clientID, pkt)
		}
	}

	if err := rc.Connect(target, cb); err != nil {
		for _, h := range s.upstreamErrorHandlers {
			h(clientID, err)
		}
	}
}

func (s *Shell) disconnectTransition(clientID string) {
	s.mu.Lock()

	ss := s.sessionLocked(clientID)
	ss.state = StateShell
	ss.reconnectBy = time.Time{}
	name := ss.client.name

	s.mu.Unlock()

	s.logf("%s (%s) disconnected from upstream", name, clientID)
	s.shutdownRemote(clientID)
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
