package codec

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
)

func (r rangeRecord) bytes() []byte {
	out := []byte{r.timestamp, r.command}
	return append(out, r.payload...)
}

func appendPreparationRecord(input, body []byte) []byte {
	input = append(input, 0, 6)
	input = binary.LittleEndian.AppendUint32(input, uint32(len(body)))
	return append(input, body...)
}

func TestParallelPreparationProtocolChange(t *testing.T) {
	input := bytes.Repeat(readCompatibilityMVD(t), 100)
	server := serverdata.Command{
		IsMVD:                true,
		ProtocolVersion:      protocol.VersionQW,
		FTEProtocolExtension: fte.ExtensionFloatCoords,
	}
	input = appendPreparationRecord(input, server.Bytes())
	player := playerInfoBytes(
		protocol.DFOrigin|protocol.DFOrigin<<1|protocol.DFOrigin<<2,
		3, 7, 4,
		[3]uint32{0x3f800000, 0x40000000, 0x40400000},
		[3]uint16{}, [4]byte{},
	)
	floatRecord := appendPreparationRecord(nil, player)
	repeats := 2 * rangeChunkTargetSize / len(floatRecord)
	input = append(input, bytes.Repeat(floatRecord, repeats)...)
	var chunks []parsedRangeChunk
	if err := parseIntoChunks(input, func(chunk parsedRangeChunk) error {
		chunks = append(chunks, chunk)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 4 {
		t.Fatal("fixture needs multiple chunks")
	}
	var sizes [5]int
	var reconstructed bytes.Buffer
	// Delay all preparation until parsing has advanced through the protocol
	// change: earlier commands must retain their original coordinate width.
	for _, chunk := range chunks {
		prepared, err := chunk.prepare()
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range prepared.records {
			reconstructed.Write(record.bytes())
			countPlayerCoordSizes(record.operations, &sizes)
		}
	}
	if !bytes.Equal(reconstructed.Bytes(), input) || sizes[2] == 0 || sizes[4] == 0 {
		t.Fatal("deferred preparation lost record bytes or protocol widths")
	}
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := encodeWithWorkers(input, resources, FormatVersion, 1)
	if err != nil {
		t.Fatal(err)
	}
	parallel, err := encodeWithWorkers(input, resources, FormatVersion, 8)
	if err != nil || !bytes.Equal(serial, parallel) {
		t.Fatal("parallel preparation differs", err)
	}
	decoded, err := Decode(parallel)
	if err != nil || !bytes.Equal(decoded, input) {
		t.Fatal("protocol-change round trip differs", err)
	}
}

func countPlayerCoordSizes(operations []rangeOperation, sizes *[5]int) {
	for _, operation := range operations {
		if operation.player != nil {
			sizes[operation.player.coordSize]++
		}
	}
}

func TestParallelPreparationErrors(t *testing.T) {
	// A valid MVD packet whose entity count cannot fit the MVZ grammar.
	body := []byte{protocol.SVCPacketEntities}
	for range maxPacketEntityCount + 1 {
		body = binary.LittleEndian.AppendUint16(body, protocol.URemove|1)
	}
	body = binary.LittleEndian.AppendUint16(body, 0)
	bad := appendPreparationRecord(nil, body)
	seed := readCompatibilityMVD(t)
	prefix := bytes.Repeat(seed, 150)
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range [][]byte{nil, {0}, bytes.Repeat(seed, 100)} {
		input := append(bytes.Clone(prefix), bad...)
		input = append(input, suffix...)
		var want string
		for _, workers := range []int{1, 2, 8} {
			encoded, err := encodeWithWorkers(
				input, resources, FormatVersion, workers,
			)
			if err == nil || encoded != nil ||
				!strings.Contains(err.Error(), "prepare MVD record 7200:") {
				t.Fatalf("%d workers: wrong preparation error: %v", workers, err)
			}
			if workers == 1 {
				want = err.Error()
			} else if err.Error() != want {
				t.Fatalf("%d workers: %v, want %s", workers, err, want)
			}
		}
	}
}
