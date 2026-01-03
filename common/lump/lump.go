package lump

import (
	"errors"

	"github.com/osm/quake/common/lump/miptex"
	"github.com/osm/quake/common/lump/qpic"
	"github.com/osm/quake/common/lump/rgba"
)

const (
	Palette = 64
	QTex    = 65
	QPic    = 66
	Sound   = 67
	MipTex  = 68
)

var (
	ErrBadLmpFormat = errors.New("unable to determine format of lump file")
)

type LumpData interface {
	Bytes() []byte
	ToRGBAImage() *rgba.Image
}

func Parse(typ uint8, data []byte) (LumpData, error) {
	switch typ {
	case QPic:
		return qpic.Parse(data)
	case MipTex:
		return miptex.Parse(miptex.DefaultWidth, miptex.DefaultHeight, data)
	}

	return nil, ErrBadLmpFormat
}
