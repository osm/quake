package proxyshell

import (
	"strconv"

	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/protocol"
)

func cloneInfoString(in *infostring.InfoString) *infostring.InfoString {
	if in == nil {
		return nil
	}

	out := &infostring.InfoString{
		Info: make([]infostring.Info, len(in.Info)),
	}

	copy(out.Info, in.Info)

	return out
}

func cloneConnectCommand(in *connect.Command) *connect.Command {
	if in == nil {
		return nil
	}

	out := *in
	out.UserInfo = cloneInfoString(in.UserInfo)

	if len(in.Extensions) > 0 {
		out.Extensions = append([]*protocol.Extension(nil), in.Extensions...)
	}

	return &out
}

func buildUpstreamConnect(identity identity) *connect.Command {
	userInfo := cloneInfoString(identity.userInfo)
	if userInfo == nil {
		userInfo = infostring.New()
	}

	userInfo.Set("name", fallbackString(identity.name, "player"))
	userInfo.Set("team", fallbackString(identity.team, "none"))
	userInfo.Set("topcolor", strconv.Itoa(int(identity.topColor)))
	userInfo.Set("bottomcolor", strconv.Itoa(int(identity.bottomColor)))

	if identity.clientVersion != "" {
		userInfo.Set("*client", identity.clientVersion)
	}

	if identity.spectator {
		userInfo.Set("spectator", "1")
	}

	return &connect.Command{
		Command:  "connect",
		Version:  "28",
		QPort:    0,
		UserInfo: userInfo,
	}
}

func fallbackString(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
