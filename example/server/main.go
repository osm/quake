package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/server"
	"github.com/osm/quake/server/quake"
)

func main() {
	addrPort := flag.String("listen-addr", "localhost:27500", "listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	srv := quake.New(logger)

	go func() {
		reader := bufio.NewReader(os.Stdin)

		for {
			input, _ := reader.ReadString('\n')

			srv.Enqueue([]command.Command{
				&print.Command{
					ID:     3,
					String: fmt.Sprintf("console: %s", input),
				},
			})
		}
	}()

	srv.HandleFunc(func(client server.Client, packet packet.Packet) []command.Command {
		gameData, ok := packet.(*clc.GameData)
		if !ok {
			return []command.Command{}
		}

		var cmds []command.Command

		for _, cmd := range gameData.Commands {
			switch c := cmd.(type) {
			case *stringcmd.Command:
				if !strings.HasPrefix(c.String, "say") {
					continue
				}

				out := fmt.Sprintf("%s: %s\n", client.GetName(), c.String[4:])
				cmds = append(cmds, &print.Command{ID: 3, String: out})
				logger.Printf("%s", out)
			}
		}

		return cmds
	})

	logger.Printf("listening on %s", *addrPort)
	if err := srv.ListenAndServe(*addrPort); err != nil {
		logger.Fatal(err)
	}
}
