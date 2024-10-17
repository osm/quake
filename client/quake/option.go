package quake

import "log"

type Option func(*Client)

func WithName(name string) Option {
	return func(c *Client) {
		c.name = name
	}
}

func WithTeam(team string) Option {
	return func(c *Client) {
		c.team = team
	}
}

func WithSpectator(isSpectator bool) Option {
	return func(c *Client) {
		c.isSpectator = isSpectator
	}
}

func WithClientVersion(clientVersion string) Option {
	return func(c *Client) {
		c.clientVersion = clientVersion
	}
}

func WithServerAddr(addrPort string) Option {
	return func(c *Client) {
		c.addrPort = addrPort
	}
}

func WithPing(ping int16) Option {
	return func(c *Client) {
		c.ping = ping
	}
}

func WithBottomColor(color byte) Option {
	return func(c *Client) {
		c.bottomColor = color
	}
}

func WithTopColor(color byte) Option {
	return func(c *Client) {
		c.topColor = color
	}
}

func WithFTEExtensions(extensions uint32) Option {
	return func(c *Client) {
		c.fteEnabled = true
		c.fteExtensions = extensions
	}
}

func WithFTE2Extensions(extensions uint32) Option {
	return func(c *Client) {
		c.fte2Enabled = true
		c.fte2Extensions = extensions
	}
}

func WithMVDExtensions(extensions uint32) Option {
	return func(c *Client) {
		c.mvdEnabled = true
		c.mvdExtensions = extensions
	}
}

func WithZQuakeExtensions(extensions uint32) Option {
	return func(c *Client) {
		c.zQuakeEnabled = true
		c.zQuakeExtensions = extensions
	}
}

func WithLogger(logger *log.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}
