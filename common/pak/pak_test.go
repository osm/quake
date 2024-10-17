package pak

import (
	"bytes"
	"encoding/base64"
	"testing"
)

const pakData = `
UEFDSxUAAADAAAAAZm9vYmFyYmF6Zm9vLnR4dAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMAAAAAwAAAGJhci50eHQAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADwAAAAMAAABiYXoudHh0AAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABIAAAADAAAA
`

func TestPak(t *testing.T) {
	data, _ := base64.StdEncoding.DecodeString(pakData)

	pak, err := Parse(data)

	if err != nil {
		t.Errorf("error when parsing pak data, %v", err)
	}

	if !bytes.Equal(data, pak.Bytes()) {
		t.Errorf("serialized pak doesn't match original pak file")
	}
}
