package crc

import (
	"testing"
)

type crcTest struct {
	name     string
	input    []byte
	seq      int
	expected byte
}

var crcTests = []crcTest{
	{
		name:     "sequence 0",
		input:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		seq:      0,
		expected: 0x2f,
	},
	{
		name:     "sequence 1",
		input:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		seq:      1,
		expected: 0x2d,
	},
	{
		name:     "sequence 2",
		input:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		seq:      2,
		expected: 0x37,
	},
}

func TestCRCs(t *testing.T) {
	for _, ct := range crcTests {
		t.Run(ct.name, func(t *testing.T) {
			crc := Byte(ct.input, ct.seq)

			if crc != ct.expected {
				t.Errorf("crc byte didn't match expected")
				t.Logf("output: %#v", crc)
				t.Logf("expected: %#v", ct.expected)
			}

		})
	}
}
