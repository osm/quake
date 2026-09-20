package frequency

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestSetV1(t *testing.T) {
	profiles, err := SetV1()
	if err != nil {
		t.Fatalf("load frequency set v1: %v", err)
	}
	tests := []struct {
		name string
		id   byte
		data []byte
		want string
	}{
		{
			name: "profile 1",
			id:   Profile1,
			data: setV1Profile1Data,
			want: "3f0e1613edd660ffd346c792b83c4edc138dd330bb7ab7644e84df2493697e33",
		},
		{
			name: "profile 2",
			id:   Profile2,
			data: setV1Profile2Data,
			want: "8bc59aeff6a6f47c06a034769b9e40c7b3e06582a4229ebf92efb564f5963ef9",
		},
		{
			name: "profile 3",
			id:   Profile3,
			data: setV1Profile3Data,
			want: "5a410a07ee648868b5d5cb08feeeb389fba9e30d18332154863d850c56287762",
		},
		{
			name: "profile 4",
			id:   Profile4,
			data: setV1Profile4Data,
			want: "f1132a52388b0cb5e461d2c08b4478f036a297e3a270fc736f1bf07ceeb8dcd9",
		},
	}
	if len(profiles) != len(tests) {
		t.Fatalf("profiles %d, want %d", len(profiles), len(tests))
	}
	wantSize := modelDataHeaderSize + int(ModelCount)*Symbols*4
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if len(test.data) != wantSize {
				t.Fatalf("profile size %d, want %d", len(test.data), wantSize)
			}
			got := fmt.Sprintf("%x", sha256.Sum256(test.data))
			if got != test.want {
				t.Fatalf("profile checksum %s, want %s", got, test.want)
			}
			if profiles[i].ID != test.id {
				t.Fatalf("profile ID %d, want %d", profiles[i].ID, test.id)
			}
			if profiles[i].Models == nil {
				t.Fatal("profile has nil models")
			}
			encoded, err := profiles[i].Models.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, test.data) {
				t.Fatal("reserialized profile differs from the embedded artifact")
			}
		})
	}
}
