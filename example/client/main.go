package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/osm/quake/client"
	"github.com/osm/quake/client/qtv"
	"github.com/osm/quake/client/quake"
	"github.com/osm/quake/common/ascii"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/qtvstringcmd"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/svc"
)

func main() {
	addrPort := flag.String("addr", "127.0.0.1:27500", "address and port to connect to")
	name := flag.String("name", "", "name")
	team := flag.String("team", "", "team")
	flag.Parse()

	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)

	if *addrPort == "" {
		logger.Fatalf("-addr is required")
	}

	var client client.Client
	var err error
	if strings.Contains(*addrPort, "@") {
		client, err = qtv.New(*name, *team, []qtv.Option{qtv.WithLogger(logger)}...)
	} else {
		client, err = quake.New(
			*name,
			*team,
			[]quake.Option{
				quake.WithSpectator(true),
				quake.WithLogger(logger),
			}...,
		)
	}
	if err != nil {
		logger.Fatalf("unable to create new client, %v", err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalCh
		client.Quit()
		os.Exit(0)
	}()

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			input, _ := reader.ReadString('\n')

			if _, ok := client.(*quake.Client); ok {
				client.Enqueue([]command.Command{
					&stringcmd.Command{String: strings.TrimSpace(input)},
				})
			} else {
				client.Enqueue([]command.Command{
					&qtvstringcmd.Command{String: strings.TrimSpace(input)},
				})
			}
		}
	}()

	client.HandleFunc(func(packet packet.Packet) []command.Command {
		gameData, ok := packet.(*svc.GameData)
		if !ok {
			return []command.Command{}
		}

		for _, cmd := range gameData.Commands {
			switch c := cmd.(type) {
			case *print.Command:
				fmt.Printf("%s", ascii.Parse(c.String))
			}
		}

		return nil
	})

	logger.Printf("connecting to %s", *addrPort)
	if err := client.Connect(*addrPort); err != nil {
		logger.Fatal(err)
	}
}
