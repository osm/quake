package miptex

import (
	"errors"

	"github.com/osm/quake/common/lump/rgba"
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

func (m *MipTex) Bytes() []byte {
	return m.Pixels
}

func (m *MipTex) ToRGBAImage() *rgba.Image {
	return rgba.ToImage(m.Width, m.Height, m.Pixels)
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

func FromPNG(data []byte) (*MipTex, error) {
	rgba, err := rgba.FromPNG(data)
	if err != nil {
		return nil, err
	}

	return &MipTex{
		Width:  rgba.Width,
		Height: rgba.Height,
		Pixels: rgba.Pixels,
	}, nil
}
