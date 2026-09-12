package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/server"
	"github.com/osm/quake/server/quake"
	"github.com/osm/quake/ui"
)

type adapter struct {
	mu      sync.Mutex
	clients map[server.Client]*ui.Connection
	factory func(server.Client) *ui.Session
	options ui.ConnectionOptions
}

// Attach after other output handlers so the menu respects the final packet size.
func Attach(s *quake.Server, factory func(server.Client) *ui.Session, options ui.ConnectionOptions) error {
	if s == nil || factory == nil {
		return fmt.Errorf("ui/server: nil server or factory")
	}
	if _, err := ui.NewConnection(ui.New(nil), options); err != nil {
		return err
	}

	a := &adapter{
		clients: make(map[server.Client]*ui.Connection),
		factory: factory,
		options: options,
	}
	s.HandleInputFunc(a.input)
	s.HandleOutputFunc(a.output)
	s.HandleDisconnectFunc(a.disconnect)
	return nil
}

func (a *adapter) connection(client server.Client) *ui.Connection {
	a.mu.Lock()
	defer a.mu.Unlock()

	if lifecycle, ok := client.(interface{ Done() <-chan struct{} }); ok {
		select {
		case <-lifecycle.Done():
			return nil
		default:
		}
	}
	if c := a.clients[client]; c != nil {
		return c
	}

	session := a.factory(client)
	if session == nil {
		session = ui.New(nil)
	}
	c, _ := ui.NewConnection(session, a.options)
	a.clients[client] = c
	return c
}

func (a *adapter) input(client server.Client, game *clc.GameData) {
	c := a.connection(client)
	if c == nil {
		return
	}

	errs := c.Input(game)
	if a.options.OnError == nil {
		return
	}
	for _, err := range errs {
		a.options.OnError(err)
	}
}

func (a *adapter) output(client server.Client, game *svc.GameData) {
	c := a.connection(client)
	if c == nil {
		return
	}

	c.Output(game, time.Now())
}

func (a *adapter) disconnect(client server.Client) {
	a.mu.Lock()
	delete(a.clients, client)
	a.mu.Unlock()
}
