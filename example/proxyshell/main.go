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
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/proxyshell"
	"github.com/osm/quake/server"
	"github.com/osm/quake/server/quake"
)

type proxyShell struct {
	log *log.Logger
	srv *quake.Server
	sh  *proxyshell.Shell

	mu     sync.Mutex
	banner map[string]bool
	queue  map[string][]command.Command
}

func main() {
	listenAddr := flag.String("listen-addr", "127.0.0.1:27501", "listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	srv := quake.New(logger)
	ps := &proxyShell{
		log:    logger,
		srv:    &srv,
		sh:     proxyshell.New(logger),
		banner: map[string]bool{},
		queue:  map[string][]command.Command{},
	}

	ps.sh.OnShellPacket(ps.handleShellPacket)
	ps.sh.OnUpstreamPacket(ps.handleUpstreamPacket)
	ps.sh.OnUpstreamError(ps.handleUpstreamError)
	ps.sh.OnShellInjectPacket(func(clientID string, pkt packet.Packet) {
		game, ok := pkt.(*svc.GameData)
		if !ok {
			return
		}
		ps.enqueueClientCommands(clientID, game.Commands...)
	})
	ps.sh.OnShellResetClient(func(clientID string) {
		ps.srv.ResetClient(clientID)
	})

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
			cmds := p.sh.ProcessShellGameData(clientID, localPkt)
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
	reset, hasTarget := p.sh.ProcessShellConnect(clientID, cmd)
	if reset {
		p.mu.Lock()
		p.banner[clientID] = false
		p.queue[clientID] = nil
		p.mu.Unlock()
	}
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

func (p *proxyShell) handleShellPacket(
	clientID string,
	pkt packet.Packet,
) {
	game, ok := pkt.(*clc.GameData)
	if !ok {
		return
	}

	for _, raw := range game.Commands {
		cmd, ok := raw.(*stringcmd.Command)
		if !ok {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(cmd.String), "new") {
			p.handleShellNew(clientID)
			continue
		}

		p.log.Printf("shell stringcmd: %q", cmd.String)
		_, _ = p.handleShellCommand(clientID, cmd.String)
	}
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
		ok, drop := p.sh.ConnectUpstream(clientID, target)
		if !ok {
			p.enqueueClientCommands(
				clientID,
				&print.Command{ID: 2, String: "already connecting\n"},
			)
			return "", false
		}

		p.enqueueClientCommands(
			clientID,
			&print.Command{ID: 2, String: "Connecting to " + target + "\n"},
		)
		p.mu.Lock()
		p.banner[clientID] = false
		p.mu.Unlock()
		if drop {
			return "drop", true
		}
		return "", false
	case "disconnect":
		p.mu.Lock()
		p.banner[clientID] = false
		p.mu.Unlock()
		if p.sh.DisconnectUpstream(clientID) {
			return "drop", true
		}
		return "", false
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
	cmds = append(cmds, game.Commands...)
	if len(local) > 0 {
		cmds = append(cmds, local...)
	}

	p.srv.WriteRawToClient(clientID, &svc.GameData{
		Seq:      game.Seq,
		Ack:      game.Ack,
		Commands: cmds,
	})
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
