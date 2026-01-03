package qpic

import (
	"bytes"
	"encoding/base64"
	"testing"
)

const qpicData = `
GAAAABgAAACuYHITYGBgERFgExNgYBERETMCcnJgEWATra6sq6uqqaempKSmp6aoqaqrq6yu
rRMSrayrE66ura2tra2tra2urq2uE6usrRMSrautrq1zc3Nzra2trXNzc3Otrq2rrRITrKyu
cnV3eHl5eHd3eHl5eHd1cq6srBMSrK2uc3VzdHh7fXt7fXt4dHN1c66trBMSrK1xdnMTE3Jz
dXd4dXNyExNzdnETrBMTFK1ydXNzEhMRFHR0FBETEnNzdXITFBITrRN0d3Z0c3N0dXp5dXRz
c3R2d3QTrRISExN1eHp4d3d3eKN8eHd3d3h6eHUTExMTrhN0dnl6enh2eqN+enZ5enp5dnQT
rhITrnJzdXZ6qHVzdHZ2dHN1qHp2dXNyrhITrXJ0dXR4dnWqc3Jyc6p1dnh0dXRyrROtq3J1
qnR2dHSpqqurqql0dHZ0qnVyq60Tq6l2qnOrc6t1dnZ2dnWrc6tzqnapqxMTrKmmqXOqchN2
eXt7eXYTcqpzqaaprBIRRqynpKl4c2CrqKenqKtgc3ippKesRhFDRkqqqXZ5dXZ4eXp6eXh2
dXl2qqlKRkMSi0qqrKt2qXd4eHh4eHh3qXarrKpKixIRrayrrXOsq6l5eKmpeHmpq6xzraus
rRERrq6urnNyrauqdaurdaqrra1zrq6urq4Srq4ScnNyca1yrKysrHKtca1zchKurq4TERER
zhERERERERERERERERERERERERHOERAQEREREREREREREREREc7OEBHOERM=
`

func TestQPic(t *testing.T) {
	data, _ := base64.StdEncoding.DecodeString(qpicData)

	qpic, err := Parse(data)

	if err != nil {
		t.Errorf("error when parsing qpic data, %v", err)
	}

	if !bytes.Equal(data, qpic.Bytes()) {
		t.Errorf("serialized qpic doesn't match original qpic file")
	}
}
