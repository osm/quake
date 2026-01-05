package miptex

import (
	"errors"

	"github.com/osm/quake/common/lump/image"
	"github.com/osm/quake/common/lump/typ"
)

const (
	DefaultWidth  = 128
	DefaultHeight = 128
	DataSize      = DefaultWidth * DefaultHeight
)

var (
	ErrInvalidSize = errors.New("invalid lump size")
)

type MipTex struct {
	Width  int
	Height int
	Pixels []byte
}

func (m *MipTex) Type() typ.Type {
	return typ.MipTex
}

func (m *MipTex) Bytes() []byte {
	return m.Pixels
}

func Parse(width, height int, data []byte) (*MipTex, error) {
	if len(data) != DataSize {
		return nil, ErrInvalidSize
	}

	return &MipTex{
		Width:  width,
		Height: height,
		Pixels: data,
	}, nil
}

func (m *MipTex) ToPNG() ([]byte, error) {
	return image.ToPNG(m.Width, m.Height, m.Pixels)
}

func FromPNG(data []byte) (*MipTex, error) {
	img, err := image.FromPNG(data)
	if err != nil {
		return nil, err
	}

	return &MipTex{
		Width:  img.Width,
		Height: img.Height,
		Pixels: img.Pixels,
	}, nil
}
