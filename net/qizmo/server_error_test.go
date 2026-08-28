package qizmo

import (
	"encoding/binary"
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
)

func TestServerEncoderErrorDoesNotConsumeResync(t *testing.T) {
	endpoint := NewEndpointState()
	endpoint.requestS2CResync()
	encoder := newServerEncoder(&freq.Tables{}, assets.Assets{}, endpoint)

	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet,
		protocol.SVCPlayerInfo, 0, 0, 0,
		100, 0, 0, 0, 0, 0, 1,
		protocol.SVCPacketEntities, 0, 0,
		protocol.SVCNOP,
	)
	if _, err := encoder.Encode(packet); err == nil {
		t.Fatal("Encode succeeded with empty frequency tables")
	}
	if !endpoint.needsS2CResync() {
		t.Fatal("failed packet consumed the resync request")
	}
}
