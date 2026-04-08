package server

import (
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command"
)

type HandlerResult struct {
	Commands []command.Command
	Consume  bool
}

type Server interface {
	ListenAndServe(string) error
	HandleFunc(h func(Client, packet.Packet) HandlerResult)
	Enqueue(cmds []command.Command)
}
