package qizmo

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
)

const (
	clientStringEscape      = 0xff
	clientProxyTokenStart   = 32
	clientCommonTokenStart  = 64
	clientDynamicTokenCount = protocol.QWMaxClients
)

var errUnsupportedClientString = errors.New("qizmo client string cannot be compressed")

func (d *ClientDecoder) decodeClientLiteralString(reader *bitReader) ([]byte, error) {
	var decoded []byte
	for {
		value, err := d.readSymbol(reader, freq.CLCStringByte)
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, value)
		if value == 0 {
			return decoded, nil
		}
	}
}

func (e *ClientEncoder) encodeClientLiteralString(writer *bitWriter, data []byte) error {
	for _, value := range data {
		if err := e.writeSymbol(writer, freq.CLCStringByte, value); err != nil {
			return err
		}
	}
	return nil
}

func (d *ClientDecoder) decodeClientString(reader *bitReader) ([]byte, error) {
	var decoded []byte
	for {
		value, err := d.readSymbol(reader, freq.CLCStringByte)
		if err != nil {
			return nil, err
		}
		if value == clientStringEscape {
			token, err := d.readSymbol(reader, freq.CLCStringToken)
			if err != nil {
				return nil, fmt.Errorf("decode dictionary token: %w", err)
			}
			text, ok := d.clientString(token)
			if !ok {
				if token < clientDynamicTokenCount {
					return nil, fmt.Errorf("dynamic Qizmo client string token %d has no player name", token)
				}
				return nil, fmt.Errorf("invalid Qizmo client string token %d", token)
			}
			decoded = append(decoded, text...)
			continue
		}

		decoded = append(decoded, value)
		if value == 0 {
			return decoded, nil
		}
	}
}

func (e *ClientEncoder) encodeClientString(writer *bitWriter, data []byte) error {
	for len(data) != 0 {
		if token, size, ok := matchClientString(data, &e.endpoint.playerNames); ok {
			if err := e.writeSymbol(writer, freq.CLCStringByte, clientStringEscape); err != nil {
				return err
			}
			if err := e.writeSymbol(writer, freq.CLCStringToken, token); err != nil {
				return err
			}
			data = data[size:]
			continue
		}

		value := data[0]
		if value == clientStringEscape {
			return errUnsupportedClientString
		}
		if err := e.writeSymbol(writer, freq.CLCStringByte, value); err != nil {
			return err
		}
		data = data[1:]
	}
	return nil
}

func (d *ClientDecoder) clientString(token byte) (string, bool) {
	if token < clientDynamicTokenCount {
		name := d.endpoint.playerNames[token]
		return name, name != ""
	}
	return staticClientString(token)
}

func staticClientString(token byte) (string, bool) {
	switch {
	case token >= clientProxyTokenStart &&
		int(token-clientProxyTokenStart) < len(assets.ClientProxyStrings):
		return assets.ClientProxyStrings[token-clientProxyTokenStart], true
	case token >= clientCommonTokenStart:
		return assets.ClientCommonStrings[token-clientCommonTokenStart], true
	default:
		return "", false
	}
}

// Qizmo searches the proxy, dynamic player-name, and common dictionaries in
// that order. It chooses the longest matching prefix within the first
// dictionary containing a match.
func matchClientString(
	data []byte,
	playerNames *[clientDynamicTokenCount]string,
) (byte, int, bool) {
	if token, size, ok := longestClientStringMatch(
		data,
		assets.ClientProxyStrings,
		clientProxyTokenStart,
	); ok {
		return token, size, true
	}
	if token, size, ok := longestClientStringMatch(data, playerNames[:], 0); ok {
		return token, size, true
	}
	return longestClientStringMatch(data, assets.ClientCommonStrings, clientCommonTokenStart)
}

func longestClientStringMatch(data []byte, dictionary []string, firstToken byte) (byte, int, bool) {
	bestToken := firstToken
	best := -1
	for index, text := range dictionary {
		if text != "" && len(text) > best && bytes.HasPrefix(data, []byte(text)) {
			best = len(text)
			bestToken = byte(int(firstToken) + index)
		}
	}
	return bestToken, best, best >= 0
}
