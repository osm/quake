package ui

import (
	"fmt"
	"time"
)

type ConnectionOptions struct {
	Namespace       string
	RefreshInterval time.Duration
	MaxPacketBytes  int
	OpenOnConnect   bool
	AlwaysVisible   bool
	OnError         func(error)
}

func (options ConnectionOptions) normalized() (ConnectionOptions, error) {
	if options.Namespace == "" {
		options.Namespace = "proxy:menu"
	}
	if _, _, err := ParseCommand("", options.Namespace); err != nil {
		return options, err
	}

	if options.RefreshInterval == 0 {
		options.RefreshInterval = 500 * time.Millisecond
	}
	if options.RefreshInterval < 0 {
		return options, fmt.Errorf("ui: negative refresh interval")
	}

	if options.MaxPacketBytes == 0 {
		options.MaxPacketBytes = 1450
	}
	if options.MaxPacketBytes < 1024 || options.MaxPacketBytes > 65507 {
		return options, fmt.Errorf("ui: packet limit must be between 1024 and 65507")
	}

	return options, nil
}
