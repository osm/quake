package codec

import (
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
)

func encodeEntityFlags(
	sink symbolSink,
	state *rangeEntityState,
	layout rangeEntityLayout,
) error {
	wantsMore := layout.header&protocol.UMoreBits != 0
	if !wantsMore {
		if len(layout.flags) != 0 {
			return fmt.Errorf("packet-entity flags lack more-bits header")
		}
		return nil
	}
	if len(layout.flags) == 0 {
		return fmt.Errorf("packet-entity more-bits header lacks flag byte")
	}
	if err := encodeEntityFlagByte(
		sink, state, layout.flags, 0,
	); err != nil {
		return err
	}
	if layout.flags[0]&fte.UEvenMore == 0 {
		if len(layout.flags) != 1 {
			return fmt.Errorf("packet-entity flag chain has %d bytes, want 1", len(layout.flags))
		}
		return nil
	}
	hasEvenMore := byte(0)
	if len(layout.flags) > 1 {
		hasEvenMore = 1
	}
	if err := sink.Put(frequency.ModelEntityEvenMorePresent, hasEvenMore); err != nil {
		return err
	}
	if hasEvenMore == 0 {
		return nil
	}
	if err := encodeEntityFlagByte(
		sink, state, layout.flags, 1,
	); err != nil {
		return err
	}
	wantsYetMore := layout.flags[1]&fte.UYetMore != 0
	if wantsYetMore != (len(layout.flags) == 3) {
		return fmt.Errorf(
			"packet-entity flag chain has invalid yet-more byte count %d",
			len(layout.flags),
		)
	}
	if wantsYetMore {
		return encodeEntityFlagByte(
			sink, state, layout.flags, 2,
		)
	}
	return nil
}

func decodeEntityFlags(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeEntityState,
	header uint16,
) ([]byte, error) {
	if header&protocol.UMoreBits == 0 {
		return nil, nil
	}
	first, err := decodeEntityFlagByte(
		decoder, models, state, 0,
	)
	if err != nil {
		return nil, err
	}
	flags := []byte{first}
	if first&fte.UEvenMore == 0 {
		return flags, nil
	}
	hasEvenMore, err := decodeSymbol(decoder, models, frequency.ModelEntityEvenMorePresent)
	if err != nil {
		return nil, err
	}
	if hasEvenMore > 1 {
		return nil, fmt.Errorf(
			"invalid decoded packet-entity even-more marker %d",
			hasEvenMore,
		)
	}
	if hasEvenMore == 0 {
		return flags, nil
	}
	second, err := decodeEntityFlagByte(
		decoder, models, state, 1,
	)
	if err != nil {
		return nil, err
	}
	flags = append(flags, second)
	if second&fte.UYetMore == 0 {
		return flags, nil
	}
	third, err := decodeEntityFlagByte(
		decoder, models, state, 2,
	)
	if err != nil {
		return nil, err
	}
	return append(flags, third), nil
}

func encodeEntityFlagByte(
	sink symbolSink,
	state *rangeEntityState,
	flags []byte,
	index int,
) error {
	flag := flags[index]
	if err := sink.Put(entityFlagModels[index], flag^state.flags[index]); err != nil {
		return err
	}
	state.flags[index] = flag
	return nil
}

func decodeEntityFlagByte(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeEntityState,
	index int,
) (byte, error) {
	xor, err := decodeSymbol(decoder, models, entityFlagModels[index])
	if err != nil {
		return 0, err
	}
	flag := xor ^ state.flags[index]
	state.flags[index] = flag
	return flag, nil
}
