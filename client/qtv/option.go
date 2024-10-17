package qtv

import "log"

type Option func(*Client)

func WithLogger(logger *log.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}
