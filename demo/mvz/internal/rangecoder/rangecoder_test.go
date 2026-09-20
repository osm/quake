package rangecoder

import (
	"math"
	"math/rand"
	"testing"

	"github.com/osm/quake/demo/mvz/frequency"
)

// Exercise the arithmetic coder independently of the MVD grammar, including
// rare symbols and abrupt model changes in every embedded profile.
func TestProfileRoundTrip(t *testing.T) {
	profiles, err := frequency.SetV1()
	if err != nil {
		t.Fatal(err)
	}
	random := rand.New(rand.NewSource(0x6d767a))
	data := make([]byte, 3*65536)
	if _, err := random.Read(data); err != nil {
		t.Fatal(err)
	}
	for _, profile := range profiles {
		checkProfileRoundTrip(t, profile.Models, data)
	}
}

func FuzzProfileRoundTrip(f *testing.F) {
	profiles, err := frequency.SetV1()
	if err != nil {
		f.Fatal(err)
	}
	random := rand.New(rand.NewSource(0x6d767a))
	seed := make([]byte, 3*64)
	if _, err := random.Read(seed); err != nil {
		f.Fatal(err)
	}
	for i := range profiles {
		f.Add(byte(i), seed)
	}
	f.Fuzz(func(t *testing.T, profile byte, data []byte) {
		if len(data) > 3*4096 {
			t.Skip()
		}
		checkProfileRoundTrip(t, profiles[int(profile)%len(profiles)].Models, data)
	})
}

func checkProfileRoundTrip(t *testing.T, models *frequency.Models, data []byte) {
	t.Helper()
	encoder := NewEncoder()
	for offset := 0; offset+3 <= len(data); offset += 3 {
		model := (int(data[offset]) | int(data[offset+1])<<8) % int(frequency.ModelCount)
		symbol := uint32(data[offset+2])
		if err := encoder.EncodeSymbol(models.Cumulative[model][:], symbol); err != nil {
			t.Fatalf("encode model %d symbol %d: %v", model, symbol, err)
		}
	}
	// Rejection must leave the round trip intact even when int has only 32 bits.
	if err := encoder.EncodeSymbol(models.Cumulative[0][:], math.MaxUint32); err == nil {
		t.Fatal("encoder accepted an out-of-range symbol")
	}
	decoder := NewDecoder(encoder.Finish())
	for offset := 0; offset+3 <= len(data); offset += 3 {
		model := (int(data[offset]) | int(data[offset+1])<<8) % int(frequency.ModelCount)
		symbol, err := decoder.DecodeSymbol(models.Cumulative[model][:])
		want := uint32(data[offset+2])
		if err != nil || symbol != want {
			t.Fatalf("decode model %d: got (%d, %v), want %d", model, symbol, err, want)
		}
	}
}
