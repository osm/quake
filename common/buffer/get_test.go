package buffer

import (
	"math"
	"reflect"
	"testing"
)

type test[T any] struct {
	name     string
	input    []byte
	expected T
}

var byteTests = []test[byte]{
	{"byte test 0", []byte{0x00}, byte(0)},
	{"byte test 255", []byte{0xff}, byte(255)},
}

var int16Tests = []test[int16]{
	{"int16 test 0", []byte{0x00, 0x00}, int16(0)},
	{"int16 test -32768", []byte{0x00, 0x80}, math.MinInt16},
	{"int16 test 32767", []byte{0xff, 0x7f}, math.MaxInt16},
}

var uint16Tests = []test[uint16]{
	{"uint16 test 0", []byte{0x00, 0x00}, uint16(0)},
	{"uint16 test 65535", []byte{0xff, 0xff}, math.MaxUint16},
}

var int32Tests = []test[int32]{
	{"int32 test 0", []byte{0x00, 0x00, 0x00, 0x00}, int32(0)},
	{"int32 test -2147483648", []byte{0x00, 0x00, 0x00, 0x80}, math.MinInt32},
	{"int32 test 2147483647", []byte{0xff, 0xff, 0xff, 0x7f}, math.MaxInt32},
}

var uint32Tests = []test[uint32]{
	{"uint32 test 0", []byte{0x00, 0x00, 0x00, 0x00}, uint32(0)},
	{"uint32 test 4294967295", []byte{0xff, 0xff, 0xff, 0xff}, uint32(math.MaxUint32)},
}

var float32Tests = []test[float32]{
	{"float32 test 0", []byte{0x00, 0x00, 0x00, 0x00}, float32(0)},
	{"float32 test 1e-45", []byte{0x01, 0x00, 0x00, 0x00}, math.SmallestNonzeroFloat32},
	{"float32 test 3.4028235e+38", []byte{0xff, 0xff, 0x7f, 0x7f}, math.MaxFloat32},
}

var angle8Tests = []test[float32]{
	{"angle8 test 0", []byte{0x00}, float32(0)},
	{"angle8 test 90", []byte{0x40}, float32(90)},
	{"angle8 test -90", []byte{0xc0}, float32(-90)},
}

var angle16Tests = []test[float32]{
	{"angle16 test 0", []byte{0x00, 0x00}, float32(0)},
	{"angle16 test 90", []byte{0x00, 0x40}, float32(90)},
	{"angle16 test 180", []byte{0x00, 0x80}, float32(180)},
}

var coord16Tests = []test[float32]{
	{"coord16 test 0", []byte{0x00, 0x00}, float32(0)},
	{"coord16 test 4095", []byte{0xf8, 0x7f}, float32(4095)},
	{"coord16 test -4096", []byte{0x00, 0x80}, float32(-4096)},
}

var coord32Tests = []test[float32]{
	{"coord32 test 0", []byte{0x00, 0x00, 0x00, 0x00}, float32(0)},
	{"coord32 test 90", []byte{0x00, 0x00, 0xb4, 0x42}, float32(90)},
	{"coord32 test -90", []byte{0x00, 0x00, 0xb4, 0xc2}, float32(-90)},
}

var stringTests = []test[string]{
	{"string test foo bar baz", []byte{0x66, 0x6f, 0x6f, 0x20, 0x62, 0x61, 0x72, 0x00}, "foo bar"},
	{"string test foo bar baz", []byte{0x66, 0x6f, 0x6f, 0x0a, 0x00}, "foo\n"},
}

func runTest[T any](t *testing.T, tests []test[T], f func(*Buffer) (T, error)) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := f(New(WithData(tt.input)))

			if err != nil {
				t.Errorf("%s: should not return an error", tt.name)
			}

			if !reflect.DeepEqual(val, tt.expected) {
				t.Errorf("%s: invalid value returned", tt.name)
				t.Logf("actual: %#v", val)
				t.Logf("expected: %#v", tt.expected)
			}
		})
	}
}

func TestBuffer(t *testing.T) {
	runTest(t, byteTests, func(buf *Buffer) (byte, error) { return buf.ReadByte() })
	runTest(t, int16Tests, func(buf *Buffer) (int16, error) { return buf.GetInt16() })
	runTest(t, uint16Tests, func(buf *Buffer) (uint16, error) { return buf.GetUint16() })
	runTest(t, int32Tests, func(buf *Buffer) (int32, error) { return buf.GetInt32() })
	runTest(t, uint32Tests, func(buf *Buffer) (uint32, error) { return buf.GetUint32() })
	runTest(t, float32Tests, func(buf *Buffer) (float32, error) { return buf.GetFloat32() })
	runTest(t, angle8Tests, func(buf *Buffer) (float32, error) { return buf.GetAngle8() })
	runTest(t, angle16Tests, func(buf *Buffer) (float32, error) { return buf.GetAngle16() })
	runTest(t, coord16Tests, func(buf *Buffer) (float32, error) { return buf.GetCoord16() })
	runTest(t, coord32Tests, func(buf *Buffer) (float32, error) { return buf.GetCoord32() })
	runTest(t, stringTests, func(buf *Buffer) (string, error) { return buf.GetString() })
}
