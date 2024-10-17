package qtv

import (
	"bytes"
	"net"
	"strings"

	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/demo/mvd"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/qtvconnect"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol/qtv"
)

func (c *Client) Connect(qtvAddr string) error {
	if !strings.Contains(qtvAddr, "@") {
		return ErrUnknownAddr
	}

	parts := strings.Split(qtvAddr, "@")
	if len(parts) != 2 {
		return ErrUnknownAddr
	}
	source := parts[0]
	addrPort := parts[1]

	conn, err := net.Dial("tcp", addrPort)
	if err != nil {
		return err
	}
	c.conn = conn

	if err := c.sendConnectHeader(source); err != nil {
		return err
	}

	retryBuf := []byte{}
	buf := make([]byte, 1024*64)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			return err
		}

		s := 0
		if c.isHandshaking {
			s = bytes.Index(buf[:n], []byte("\n\n"))
			if s == -1 {
				continue
			}

			s += 2
			c.isHandshaking = false
		}

		mvdData := append(retryBuf, buf[s:n]...)

		// QTV might send incomplete data packets, in that case
		// the MVD parser will return bad read. In those cases
		// we'll store the current buffer in the retryBuf and
		// then we'll append the next chunk och data to the
		// retryBuf during the next iteration and try the
		// parsing again.
		demo, err := mvd.Parse(c.ctx, mvdData)
		if err != nil {
			retryBuf = mvdData
			continue
		}

		if len(retryBuf) > 0 {
			retryBuf = []byte{}
		}

		var cmds []command.Command

		for _, d := range demo.Data {
			switch p := d.Read.Packet.(type) {
			case *svc.GameData:
				cmds = append(cmds, c.handleGameData(p)...)

				for _, h := range c.handlers {
					cmds = append(cmds, h(p)...)
				}
			}
		}

		c.cmdsMu.Lock()
		for _, cmd := range append(cmds, c.cmds...) {
			if _, err := c.conn.Write(cmd.Bytes()); err != nil {
				c.logger.Printf("unable to write command data, %v\n", err)
			}
		}
		c.cmds = []command.Command{}
		c.cmdsMu.Unlock()
	}
}

func (c *Client) sendConnectHeader(sourceID string) error {
	c.isHandshaking = true

	_, err := c.conn.Write((&qtvconnect.Command{
		Version:    qtv.Version,
		Extensions: qtv.ExtensionDownload | qtv.ExtensionSetInfo | qtv.ExtensionUserList,
		Source:     sourceID,
		UserInfo: infostring.New(
			infostring.WithKeyValue("name", c.name),
			infostring.WithKeyValue("team", c.team),
		),
	}).Bytes())

	return err
}
