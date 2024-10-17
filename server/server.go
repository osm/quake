package server

import (
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command"
)

type Server interface {
	ListenAndServe(string) error
	HandleFunc(h func(*Client, packet.Packet) []command.Command)
	Enqueue(cmds []command.Command)
}
