package client

import (
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command"
)

type Client interface {
	Connect(addrPort string) error
	Enqueue([]command.Command)
	HandleFunc(func(packet.Packet) []command.Command)
	Quit()
}
