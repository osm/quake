package proxy

import (
	"sync"

	"github.com/osm/quake/packet/command"
)

type CommandQueue struct {
	sync.Mutex
	commands []command.Command
}

func (cq *CommandQueue) Enqueue(cmds ...command.Command) {
	cq.Lock()
	defer cq.Unlock()

	cq.commands = append(cq.commands, cmds...)
}

func (cq *CommandQueue) Dequeue() command.Command {
	cq.Lock()
	defer cq.Unlock()

	if len(cq.commands) == 0 {
		return nil
	}

	cmd := cq.commands[0]
	cq.commands = cq.commands[1:]
	return cmd
}
