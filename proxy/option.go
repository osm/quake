package proxy

import (
	"log"
	"time"
)

type Option func(*Proxy)

func WithReadDeadline(readDeadline time.Duration) Option {
	return func(p *Proxy) {
		p.readDeadline = readDeadline

	}
}

func WithLogger(logger *log.Logger) Option {
	return func(p *Proxy) {
		p.logger = logger
	}
}
