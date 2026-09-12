package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/osm/quake/server/quake"
	"github.com/osm/quake/ui"
	uiserver "github.com/osm/quake/ui/server"
)

func main() {
	addr := flag.String("listen-addr", "127.0.0.1:27500", "UDP listen address")
	mapName := flag.String("map", "dm3", "lobby map name without maps/ or .bsp (must be installed in the client)")
	originText := flag.String("origin", "564 -195 147", "fixed player origin: x y z in Quake units")
	anglesText := flag.String("angles", "29 89 0", "fixed view angles: pitch yaw roll in degrees")
	flag.Parse()

	logger := log.New(os.Stdout, "chatserver: ", log.LstdFlags)
	lobby, err := parseLobby(*mapName, *originText, *anglesText)
	if err != nil {
		logger.Fatal(err)
	}

	s := quake.New(logger)
	lobby.Stats = menuStats
	if err := s.EnableLobby(lobby); err != nil {
		logger.Fatal(err)
	}

	chat := newChat()
	s.HandleFunc(chat.handle)
	s.HandleDisconnectFunc(chat.disconnect)
	options := ui.ConnectionOptions{
		AlwaysVisible: true,
		OnError: func(err error) {
			logger.Print(err)
		},
	}
	if err := uiserver.Attach(&s, chat.menu, options); err != nil {
		logger.Fatal(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		<-signals
		_ = s.Close()
	}()

	logger.Printf("listening on %s; in Quake: connect %s", *addr, *addr)
	logger.Printf("lobby map=%s origin=%v angles=%v", lobby.Map, lobby.Origin, lobby.Angles)
	if err := s.ListenAndServe(*addr); err != nil {
		logger.Fatal(err)
	}
}
