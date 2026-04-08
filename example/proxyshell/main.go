package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/disconnect"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/proxyshell"
	"github.com/osm/quake/server"
	"github.com/osm/quake/server/quake"
)

type proxyShell struct {
	log *log.Logger
	srv *quake.Server
	sh  *proxyshell.Shell

	mu                sync.Mutex
	banner            map[string]bool
	queue             map[string][]command.Command
	disconnectPending map[string]bool
}

func main() {
	listenAddr := flag.String("listen-addr", "127.0.0.1:27501", "listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	srv := quake.New(logger)
	ps := &proxyShell{
		log:               logger,
		srv:               &srv,
		sh:                proxyshell.New(logger),
		banner:            map[string]bool{},
		queue:             map[string][]command.Command{},
		disconnectPending: map[string]bool{},
	}

	ps.sh.OnNew(ps.handleShellNew)
	ps.sh.OnString(func(clientID, s string) {
		ps.handleShellString(clientID, s)
	})
	ps.sh.OnUpstream(ps.handleUpstreamPacket)
	ps.sh.OnUpstreamError(ps.handleUpstreamError)

	srv.HandleFunc(ps.handleClientPacket)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigc
		ps.sh.Shutdown()
		_ = srv.Close()
	}()

	logger.Printf("listening on %s", *listenAddr)
	if err := srv.ListenAndServe(*listenAddr); err != nil {
		logger.Fatalf("unable to serve, %v", err)
	}
}

func (p *proxyShell) handleClientPacket(
	client server.Client,
	pkt packet.Packet,
) server.HandlerResult {
	clientID := client.GetAddr()
	inShell := p.sh.IsInShell(clientID)

	switch localPkt := pkt.(type) {
	case *clc.Connectionless:
		p.handleConnectPacket(clientID, localPkt)
	case *clc.GameData:
		if inShell {
			cmds := p.sh.HandleGameData(clientID, localPkt)
			local := p.drainClientQueue(clientID)
			if len(local) > 0 {
				cmds = append(cmds, local...)
			}
			return server.HandlerResult{Commands: cmds}
		}
		p.handleUpstreamGameData(clientID, localPkt)
		return server.HandlerResult{Consume: true}
	}

	if inShell {
		local := p.drainClientQueue(clientID)
		if len(local) > 0 {
			return server.HandlerResult{Commands: local}
		}
	}

	return server.HandlerResult{}
}

func (p *proxyShell) handleConnectPacket(
	clientID string,
	pkt *clc.Connectionless,
) {
	cmd, ok := pkt.Command.(*connect.Command)
	if !ok {
		return
	}
	if p.sh.HandleConnect(clientID, cmd) {
		p.mu.Lock()
		p.banner[clientID] = false
		p.queue[clientID] = nil
		p.mu.Unlock()
	}
	hasTarget := p.sh.AttachUpstream(clientID)
	p.mu.Lock()
	p.banner[clientID] = !hasTarget
	p.mu.Unlock()
	p.log.Printf(
		"client name=%q team=%q target=%v shell=%v",
		cmd.UserInfo.Get("name"),
		cmd.UserInfo.Get("team"),
		hasTarget,
		p.sh.IsInShell(clientID),
	)
}

func (p *proxyShell) handleShellString(clientID, s string) {
	p.log.Printf("shell stringcmd: %q", s)
	_, _ = p.handleShellCommand(clientID, s)
}

func (p *proxyShell) handleShellNew(clientID string) {
	p.mu.Lock()
	show := p.banner[clientID]
	if show {
		p.banner[clientID] = false
	}
	p.mu.Unlock()
	if !show {
		return
	}
	p.queueBanner(clientID)
}

func (p *proxyShell) queueBanner(clientID string) {
	p.enqueueClientCommands(
		clientID,
		&print.Command{ID: 2, String: "proxyshell example\n"},
		&print.Command{
			ID:     2,
			String: "use .connect <host:port> to attach upstream\n",
		},
	)
}

func (p *proxyShell) handleUpstreamGameData(
	clientID string,
	pkt *clc.GameData,
) {
	out := make([]command.Command, 0, len(pkt.Commands))
	modified := false

	for _, raw := range pkt.Commands {
		cmd, ok := raw.(*stringcmd.Command)
		if !ok {
			out = append(out, raw)
			continue
		}

		rewritten, forward := p.handleShellCommand(clientID, cmd.String)
		if !forward {
			modified = true
			continue
		}
		if rewritten != cmd.String {
			modified = true
			raw = &stringcmd.Command{String: rewritten}
		}
		out = append(out, raw)
	}

	if len(out) == 0 {
		return
	}

	if modified {
		_ = p.sh.WriteUpstream(clientID, &clc.GameData{
			Seq:      pkt.Seq,
			Ack:      pkt.Ack,
			QPort:    pkt.QPort,
			Commands: out,
		})
		return
	}

	_ = p.sh.WriteUpstream(clientID, pkt)
}

func (p *proxyShell) handleShellCommand(clientID, s string) (string, bool) {
	cmd, args, ok := parseShellCommand(s)
	if !ok {
		return s, true
	}

	switch cmd {
	case "help":
		p.enqueueClientCommands(
			clientID,
			&print.Command{
				ID:     2,
				String: "commands: .connect .disconnect .help\n",
			},
			&print.Command{
				ID:     2,
				String: "use .connect <host:port> to attach to a server\n",
			},
		)
		return "", false
	case "connect":
		if len(args) == 0 {
			p.enqueueClientCommands(
				clientID,
				&print.Command{ID: 2, String: "usage: .connect <host:port>\n"},
			)
			return "", false
		}
		target := strings.Join(args, " ")
		if !p.sh.Connect(clientID, target) {
			p.enqueueClientCommands(
				clientID,
				&print.Command{ID: 2, String: "already connecting\n"},
			)
			return "", false
		}
		p.enqueueClientCommands(
			clientID,
			&stufftext.Command{String: "disconnect;wait;wait;wait;reconnect\n"},
			&print.Command{ID: 2, String: "Connecting to " + target + "\n"},
		)
		p.flushClientQueueNow(clientID)
		return "", false
	case "disconnect":
		p.enqueueClientCommands(
			clientID,
			&stufftext.Command{String: "changing\n"},
		)
		p.mu.Lock()
		p.banner[clientID] = false
		p.disconnectPending[clientID] = true
		p.mu.Unlock()
		return "drop", true
	default:
		return s, true
	}
}

func (p *proxyShell) handleUpstreamPacket(
	clientID string,
	pkt packet.Packet,
) {
	if _, ok := pkt.(*svc.Connectionless); ok {
		return
	}

	game, ok := pkt.(*svc.GameData)
	if !ok {
		return
	}

	local := p.drainClientQueue(clientID)
	cmds := make([]command.Command, 0, len(game.Commands)+len(local))
	for _, raw := range game.Commands {
		if _, ok := raw.(*disconnect.Command); ok &&
			p.disconnectPending[clientID] {
			continue
		}
		cmds = append(cmds, raw)
	}
	if len(local) > 0 {
		cmds = append(cmds, local...)
	}

	p.srv.WriteRawToClient(clientID, &svc.GameData{
		Seq:      game.Seq,
		Ack:      game.Ack,
		Commands: cmds,
	})

	p.mu.Lock()
	pending := p.disconnectPending[clientID]
	if pending {
		p.disconnectPending[clientID] = false
	}
	p.mu.Unlock()

	if pending {
		p.sh.Disconnect(clientID)
		p.srv.ResetClient(clientID)
		p.flushClientQueueNow(clientID)
	}
}

func (p *proxyShell) handleUpstreamError(clientID string, err error) {
	if err == nil {
		return
	}
	p.log.Printf("upstream connect failed: %v", err)
	p.enqueueClientCommands(clientID, &print.Command{ID: 2, String: "connect failed\n"})
	p.flushClientQueueNow(clientID)
}

func (p *proxyShell) enqueueClientCommands(
	clientID string,
	cmds ...command.Command,
) {
	p.mu.Lock()
	p.queue[clientID] = append(p.queue[clientID], cmds...)
	p.mu.Unlock()
}

func (p *proxyShell) drainClientQueue(clientID string) []command.Command {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue[clientID]) == 0 {
		return nil
	}
	out := append([]command.Command(nil), p.queue[clientID]...)
	p.queue[clientID] = nil
	return out
}

func (p *proxyShell) flushClientQueueNow(clientID string) {
	cmds := p.drainClientQueue(clientID)
	if len(cmds) == 0 {
		return
	}
	p.srv.EnqueueToClient(clientID, cmds)
	p.srv.FlushClient(clientID)
}

func parseShellCommand(s string) (string, []string, bool) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "."):
		fields := strings.Fields(strings.TrimSpace(s[1:]))
		if len(fields) == 0 {
			return "", nil, false
		}
		return strings.ToLower(fields[0]), fields[1:], true
	case strings.HasPrefix(strings.ToLower(s), "say ."):
		fields := strings.Fields(strings.TrimSpace(s[len("say ."):]))
		if len(fields) == 0 {
			return "", nil, false
		}
		return strings.ToLower(fields[0]), fields[1:], true
	case strings.HasPrefix(strings.ToLower(s), "say \""):
		inner := strings.TrimSpace(s[len("say "):])
		inner = strings.Trim(inner, "\"")
		return parseShellCommand(inner)
	default:
		return "", nil, false
	}
}
