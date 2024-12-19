package loc

import (
	"testing"
)

func TestLoc(t *testing.T) {
	dm3Loc := []byte(`-7040 -1856 -128 sng-mega
1536 -1664 -1408 ra-tunnel
11776 -7424 -192 ya
12160 3456 -704 rl
-5056 -5440 -128 sng-ra
4096 6144 1728 lifts`)

	loc, err := Parse(dm3Loc)
	if err != nil {
		t.Errorf("error when parsing loc data, %v", err)
	}

	l := loc.Get([3]float32{1300, -700, -24})
	if l.Name != "ya" {
		t.Errorf("expected to be at ya, got %v", l.Name)
	}
}
