package lump

import (
	"errors"

	"github.com/osm/quake/common/lump/miptex"
	"github.com/osm/quake/common/lump/qpic"
	"github.com/osm/quake/common/lump/typ"
)

var (
	ErrBadLmpFormat = errors.New("unable to determine format of lump file")
)

type LumpData interface {
	Bytes() []byte
	ToPNG() ([]byte, error)
	Type() typ.Type
}

func Parse(t typ.Type, data []byte) (LumpData, error) {
	switch t {
	case typ.QPic:
		return qpic.Parse(data)
	case typ.MipTex:
		return miptex.Parse(miptex.DefaultWidth, miptex.DefaultHeight, data)
	}

	return nil, ErrBadLmpFormat
}
