package qizmo

import (
	"fmt"

	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
)

type ClientExtension struct {
	Opcode  byte
	Payload []byte
}

func (d *ClientDecoder) DecodeWithExtensions(packet []byte) ([]byte, []ClientExtension, error) {
	if !qizmoprotocol.IsCompressedLinkPacket(packet) {
		operations, err := d.observeRawPacket(packet)
		if err != nil {
			return nil, nil, fmt.Errorf("separate Qizmo client extensions: %w", err)
		}
		ordinary, extensions := splitClientExtensions(packet, operations)
		return ordinary, extensions, nil
	}

	decoded, operations, err := d.decodeCompressedPacket(packet)
	if err != nil {
		return nil, nil, err
	}
	ordinary, extensions := splitClientExtensions(decoded, operations)
	return ordinary, extensions, nil
}

func (d *ClientDecoder) observeS2CResyncRequest(operations []clientOperation) {
	for _, operation := range operations {
		if operation.opcode == qizmoprotocol.CLCRequestS2CResync {
			d.endpoint.requestS2CResync()
		}
	}
}

func splitClientExtensions(packet []byte, operations []clientOperation) ([]byte, []ClientExtension) {
	var extensions []ClientExtension
	for _, operation := range operations {
		if qizmoprotocol.IsClientExtensionOpcode(operation.opcode) {
			extensions = append(extensions, ClientExtension{
				Opcode:  operation.opcode,
				Payload: append([]byte(nil), operation.payload...),
			})
		}
	}
	if len(extensions) == 0 {
		return packet, nil
	}

	ordinary := append([]byte(nil), packet[:rawClientHeaderSize]...)
	for _, operation := range operations {
		if !qizmoprotocol.IsClientExtensionOpcode(operation.opcode) {
			ordinary = appendClientOperation(ordinary, operation)
		}
	}
	return ordinary, extensions
}
