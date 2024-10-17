package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/osm/quake/common/args"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/proxy"
)

func main() {
	addrPort := flag.String("listen-addr", "localhost:27500", "listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	prx := proxy.New(proxy.WithLogger(logger))

	prx.HandleFunc(proxy.SVC, func(_ *proxy.Client, packet packet.Packet) {
		gameData, ok := packet.(*svc.GameData)
		if !ok {
			return
		}

		for _, cmd := range gameData.Commands {
			stufftextCmd, ok := cmd.(*stufftext.Command)
			if !ok {
				continue
			}

			for _, arg := range args.Parse(stufftextCmd.String) {
				str := arg.Cmd

				if len(arg.Args) > 0 {
					str = fmt.Sprintf("%s %s", str, strings.Join(arg.Args, " "))
				}

				logger.Printf("%s\n", str)
			}
		}
	})

	logger.Printf("listening on %s", *addrPort)
	if err := prx.Serve(*addrPort); err != nil {
		logger.Fatalf("unable to serve, %v", err)
	}
}
