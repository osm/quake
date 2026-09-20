package codec

import (
	"testing"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
)

func TestRangeRejectsOutputOverflowBeforeAppend(t *testing.T) {
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		t.Fatal(err)
	}
	models := resources.profiles[0].models
	text := rangeTextOperation{
		opcode: protocol.SVCCenterPrint, payload: []byte{'a', 'b', 0},
	}
	player := rangePlayerInfo{
		raw: []byte{protocol.SVCPlayerInfo, 0, 0, 0, 0}, coordSize: 2,
	}
	tests := []struct {
		name     string
		record   rangeRecord
		trailing []byte
		limit    int
	}{
		{name: "record header", record: rangeRecord{command: 6}, limit: 1},
		{name: "body header", record: rangeRecord{command: 6, structured: true}, limit: 2},
		{
			name:   "multiple prefix",
			record: rangeRecord{command: 3, structured: true, prefix: []byte{1, 2, 3, 4}},
			limit:  6,
		},
		{
			name: "raw operation",
			record: rangeRecord{command: 6, structured: true, operations: []rangeOperation{
				{raw: []byte{1, 2, 3, 4}},
			}},
			limit: 8,
		},
		{
			name: "text operation",
			record: rangeRecord{command: 6, structured: true, operations: []rangeOperation{
				{text: &text},
			}},
			limit: 8,
		},
		{
			name: "player operation",
			record: rangeRecord{command: 6, structured: true, operations: []rangeOperation{
				{player: &player},
			}},
			limit: 10,
		},
		{
			name: "sound operation",
			record: rangeRecord{command: 6, structured: true, operations: []rangeOperation{
				{sound: &rangeSound{coordSize: 2}},
			}},
			limit: 8,
		},
		{
			name: "trailing bytes", record: rangeRecord{command: 6},
			trailing: []byte{1, 2, 3, 4}, limit: 4,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records := []rangeRecord{test.record}
			encoder := rangecoder.NewEncoder()
			sink := &rangeSymbolSink{encoder: encoder, models: models}
			if err := encodeChunkSymbols(sink, records, test.trailing); err != nil {
				t.Fatal(err)
			}
			data := collectLiterals(records, test.trailing)
			chunk := rangeChunkDecoder{
				decoder: rangecoder.NewDecoder(encoder.Finish()), models: models,
				literals: data.general, textStreams: data.text,
				decodedSize: test.limit, out: make([]byte, 0, test.limit),
			}
			err := chunk.decodeRecords()
			if err == nil {
				err = chunk.finish()
			}
			if err == nil {
				t.Fatal("decoder accepted output beyond the advertised size")
			}
			if len(chunk.out) > test.limit || cap(chunk.out) > test.limit {
				t.Fatalf("output grew before rejection: len %d, cap %d, limit %d",
					len(chunk.out), cap(chunk.out), test.limit)
			}
		})
	}
}

func TestRangeRejectsEmptyRawOperation(t *testing.T) {
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		t.Fatalf("load resources: %v", err)
	}
	models := resources.profiles[0].models
	encoder := rangecoder.NewEncoder()
	sink := &rangeSymbolSink{encoder: encoder, models: models}
	if err := sink.Put(frequency.ModelOperationKind, rangeOperationRaw); err != nil {
		t.Fatalf("encode raw operation kind: %v", err)
	}
	if err := putUvarint(sink, frequency.ModelRawByte, 0); err != nil {
		t.Fatalf("encode empty raw operation length: %v", err)
	}
	chunk := rangeChunkDecoder{
		decoder:     rangecoder.NewDecoder(encoder.Finish()),
		models:      models,
		decodedSize: 16,
	}
	if err := chunk.decodeOperations(); err == nil {
		t.Fatal("decoder accepted an empty raw operation")
	}

	operationEncoder := rangeOperationEncoder{sink: sink}
	if err := operationEncoder.encodeRawLength(nil); err == nil {
		t.Fatal("encoder accepted an empty raw operation")
	}
}

func TestPacketEntityTailRespectsDecodedRemainder(t *testing.T) {
	resources := compatibilityResources(t, FormatVersion, frequency.Profile1)
	models := resources.profiles[0].models
	encoder := rangecoder.NewEncoder()
	sink := &rangeSymbolSink{encoder: encoder, models: models}
	if err := putUvarint(sink, frequency.ModelEntityTailLength, 16); err != nil {
		t.Fatalf("encode tail length: %v", err)
	}
	decoder := rangeEntityFieldDecoder{
		decoder:      rangecoder.NewDecoder(encoder.Finish()),
		models:       models,
		decodedLimit: 8,
	}
	if err := decoder.decodeTail(); err == nil {
		t.Fatal("packet-entity decoder accepted a tail beyond the decoded remainder")
	}
}
