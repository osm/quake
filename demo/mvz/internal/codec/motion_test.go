package codec

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
)

type motionVariant struct {
	name      string
	player    bool
	coordSize int
	angleSize int
}

// Check encoded bytes as well as decoded output: matching changes to the
// encoder and decoder can round trip successfully while breaking the format.
func TestMotionPredictionCompatibility(t *testing.T) {
	goldens := map[string]string{
		"profile1/player-coord16": "c2e69cd9c1ad05fcfe15ecd773e5e141" +
			"ec63e3b10a8ef2a171ddeaf393e03136",
		"profile1/player-coord32": "0aae67e764ddaef4fee2d8411838c08c" +
			"d98a166e7a09b429c5d136b76a5dba63",
		"profile1/entity-coord16-angle8": "fb0f14d0e39fac259990c1c6226f39b1" +
			"faf6dd1242ee5988cb0a1e3a86dd6a91",
		"profile1/entity-coord16-angle16": "c919da64602ee798de9b89ee3b81299a" +
			"f022b5d4bf4ef8281c6dc3db97af053c",
		"profile1/entity-coord32-angle8": "65f596bf29e6d08c2f7924aaac6423a2" +
			"a2a626236e9e190ee21358fd40c35b8e",
		"profile1/entity-coord32-angle16": "827b0bd65a589d4528d5ca2c0d6c526d" +
			"14f6efb1344fc3803e98df733fd02fe1",
		"profile2/player-coord16": "94d7ba47972c78c1c63e30ef4bbda124" +
			"0b1a8523921409e227f5e8fb080892c2",
		"profile2/player-coord32": "24fe36f7dd03c990b453262c7938a887" +
			"2a102b352e463bf832de07b00f6c073b",
		"profile2/entity-coord16-angle8": "e98691132ba27cf0fc5fe180517c53fc" +
			"03a511c9ee0d4b4354290c6488a18f4b",
		"profile2/entity-coord16-angle16": "c56906baf2381e6cbf07a7085728dc31" +
			"4e2b076a660dd3a2b4f03b89093c59e9",
		"profile2/entity-coord32-angle8": "38e2e406314985afedef5ca9b8d40af5" +
			"82a6beae8637df8c4db627f28c7889e0",
		"profile2/entity-coord32-angle16": "09896ce14fe68d2fbed1248ff7511a57" +
			"40ab7e90e330312688bee0431367df6f",
		"profile3/player-coord16": "50ed7dd28ac48a211fc2a92742622bd0" +
			"1f42e82d26aed74b437243c63b7df082",
		"profile3/player-coord32": "1bc7a0737e43992bd67db4aa27ccb52f" +
			"a7c7bf0ece74c8e82f42fb0bbb5e535d",
		"profile3/entity-coord16-angle8": "241f1d89231ff242c1b164627e9c54c2" +
			"9cfedbc146b9dce16f4a4ec0763c6acd",
		"profile3/entity-coord16-angle16": "a8e0ce6c3fc506ee44296893f460128b" +
			"dc3577b52cb6e6d380a3134d31cf5f3a",
		"profile3/entity-coord32-angle8": "e0638b00f448a17f4ede23d0e1836b60" +
			"9b85ed8e3678160851cae9cb6007ffb8",
		"profile3/entity-coord32-angle16": "9d5ff18d040f5b20d70b7b11d7870125" +
			"31fb4b80462ded132917edb487f5fe12",
		"profile4/player-coord16": "41d0d69ce6b0236faffd10b7fe4e41f6" +
			"02788c5966dde2c53511dedbb1abf207",
		"profile4/player-coord32": "4b6f068430728cad35348b5a201cb0d3" +
			"710cac567af5fa0d08775e277e74a38a",
		"profile4/entity-coord16-angle8": "b928c26cb0b5aa6ce1a7a96ee5b0c268" +
			"b57a245a79038190b49c881bd36cfa7f",
		"profile4/entity-coord16-angle16": "e63f85cb90f94f6c68849523407c0f2f" +
			"119dca7ed1ff97e2fe08e4ca823d0d7f",
		"profile4/entity-coord32-angle8": "05a91249dfc364627f4584841a79b955" +
			"6278e8bf4a3747a136a8189bf7414f99",
		"profile4/entity-coord32-angle16": "bd50f90c8aec27d85a25bd4949a60c95" +
			"3e5f286814d5135696ad6f86ea739dc7",
	}
	variants := []motionVariant{
		{name: "player-coord16", player: true, coordSize: 2},
		{name: "player-coord32", player: true, coordSize: 4},
		{name: "entity-coord16-angle8", coordSize: 2, angleSize: 1},
		{name: "entity-coord16-angle16", coordSize: 2, angleSize: 2},
		{name: "entity-coord32-angle8", coordSize: 4, angleSize: 1},
		{name: "entity-coord32-angle16", coordSize: 4, angleSize: 2},
	}
	profiles := []byte{
		frequency.Profile1, frequency.Profile2, frequency.Profile3, frequency.Profile4,
	}
	for _, profile := range profiles {
		for _, variant := range variants {
			name := fmt.Sprintf("profile%d/%s", profile, variant.name)
			t.Run(name, func(t *testing.T) {
				checkMotionPrediction(t, profile, variant, goldens[name])
			})
		}
	}
}

func checkMotionPrediction(
	t *testing.T,
	profile byte,
	variant motionVariant,
	wantHash string,
) {
	t.Helper()
	resources := compatibilityResources(t, FormatVersion, profile)
	models := resources.profiles[0].models
	encoder := rangecoder.NewEncoder()
	var encodeState rangeCodecState
	sink := &rangeSymbolSink{encoder: encoder, models: models}
	samples := motionSamples(variant)
	for i, sample := range samples {
		encodeState.currentTime = sample.timestamp
		var err error
		if variant.player {
			err = encodePlayerInfo(sink, &encodeState, sample.player)
		} else {
			err = encodePacketEntities(sink, &encodeState, sample.packet)
		}
		if err != nil {
			t.Fatalf("encode sample %d: %v", i, err)
		}
	}
	encoded := encoder.Finish()
	got := fmt.Sprintf("%x", sha256.Sum256(encoded))
	if got != wantHash {
		t.Fatalf("encoded checksum %s, want %s", got, wantHash)
	}
	decoder := rangecoder.NewDecoder(encoded)
	var decodeState rangeCodecState
	for i, sample := range samples {
		decodeState.currentTime = sample.timestamp
		var decoded []byte
		var err error
		if variant.player {
			decoded, err = decodePlayerInfo(decoder, models, &decodeState)
		} else {
			decoded, err = decodePacketEntities(
				decoder, models, &decodeState, sample.packet.delta, maxChunkSize,
			)
		}
		if err != nil {
			t.Fatalf("decode sample %d: %v", i, err)
		}
		if !bytes.Equal(decoded, sample.raw) {
			t.Fatalf("sample %d: decoded %x, want %x", i, decoded, sample.raw)
		}
	}
}

type motionSample struct {
	timestamp uint32
	raw       []byte
	player    rangePlayerInfo
	packet    rangePacketEntities
}

func motionSamples(variant motionVariant) []motionSample {
	// Include initial and repeated timestamps, changing intervals, and wraparound
	// of clocks and field values in both prediction directions.
	times := []uint32{0, 1, 2, 7, 10, 200, 200, 201, 500, 0xfffffff0, 4, 9}
	values := []uint32{
		0, 1, 0xffff, 0x8000, 16, 17, 256, 257, 0xffffffff, 0x41200000, 0xc2480000, 3,
	}
	var samples []motionSample
	for i, timestamp := range times {
		origin := [3]uint32{
			values[i], values[(i+3)%len(values)], values[(i+7)%len(values)],
		}
		angles := [3]uint16{
			uint16(values[(i+1)%len(values)]),
			uint16(values[(i+4)%len(values)]),
			uint16(values[(i+8)%len(values)]),
		}
		sample := motionSample{timestamp: timestamp}
		if variant.player {
			bits := uint16(protocol.DFOrigin | protocol.DFOrigin<<1 | protocol.DFOrigin<<2 |
				protocol.DFAngles | protocol.DFAngles<<1 | protocol.DFAngles<<2 |
				protocol.DFModel | protocol.DFSkinNum | protocol.DFEffects | protocol.DFWeaponFrame)
			sample.raw = playerInfoBytes(
				bits, 3, byte(i), variant.coordSize, origin, angles, [4]byte{1, 2, 3, 4},
			)
			sample.player = rangePlayerInfo{raw: sample.raw, coordSize: variant.coordSize}
		} else {
			entity := rangePacketEntity{
				raw: rangePacketEntityBytes(
					5, variant.coordSize, variant.angleSize,
					[5]byte{1, 2, 3, 4, 5}, origin, angles, nil,
				),
				flagCount: 1, coordSize: variant.coordSize, angleSize: variant.angleSize,
			}
			sample.packet = rangePacketEntities{
				delta: i%2 != 0, base: byte(i * 31),
				entities: []rangePacketEntity{entity},
			}
			sample.raw = packetEntityOperationBytes(sample.packet)
		}
		samples = append(samples, sample)
	}
	return samples
}

func playerInfoBytes(
	bits uint16,
	player, frame byte,
	coordSize int,
	origin [3]uint32,
	angles [3]uint16,
	byteFields [4]byte,
) []byte {
	out := []byte{protocol.SVCPlayerInfo, player}
	out = binary.LittleEndian.AppendUint16(out, bits)
	out = append(out, frame)
	for _, value := range origin {
		if coordSize == 4 {
			out = binary.LittleEndian.AppendUint32(out, value)
		} else {
			out = binary.LittleEndian.AppendUint16(out, uint16(value))
		}
	}
	for _, value := range angles {
		out = binary.LittleEndian.AppendUint16(out, value)
	}
	return append(out, byteFields[:]...)
}

func rangePacketEntityBytes(
	number uint16,
	coordSize, angleSize int,
	byteFields [5]byte,
	origin [3]uint32,
	angle [3]uint16,
	tail []byte,
) []byte {
	header := number |
		protocol.UMoreBits |
		protocol.UFrame |
		protocol.UOrigin1 |
		protocol.UOrigin2 |
		protocol.UOrigin3 |
		protocol.UAngle2
	more := byte(protocol.UModel | protocol.UColorMap | protocol.USkin |
		protocol.UEffects | protocol.UAngle1 | protocol.UAngle3)
	out := binary.LittleEndian.AppendUint16(nil, header)
	out = append(out, more)
	out = append(out, byteFields[:]...)
	for axis := 0; axis < 3; axis++ {
		if coordSize == 2 {
			out = binary.LittleEndian.AppendUint16(out, uint16(origin[axis]))
		} else {
			out = binary.LittleEndian.AppendUint32(out, origin[axis])
		}
		if angleSize == 1 {
			out = append(out, byte(angle[axis]))
		} else {
			out = binary.LittleEndian.AppendUint16(out, angle[axis])
		}
	}
	return append(out, tail...)
}

func packetEntityOperationBytes(packet rangePacketEntities) []byte {
	operation := byte(protocol.SVCPacketEntities)
	if packet.delta {
		operation = protocol.SVCDeltaPacketEntities
	}
	out := []byte{operation}
	if packet.delta {
		out = append(out, packet.base)
	}
	for _, entity := range packet.entities {
		out = append(out, entity.raw...)
	}
	return binary.LittleEndian.AppendUint16(out, 0)
}
