package qpic

import (
	"errors"
	"fmt"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/lump/rgba"
)

const (
	headerSize = 8
)

var (
	ErrInvalidSize = errors.New("invalid lump size")
)

type QPic struct {
	Width  int
	Height int
	Pixels []byte
}

func (q *QPic) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint32(uint32(q.Width))
	buf.PutUint32(uint32(q.Height))
	buf.PutBytes(q.Pixels)

	return buf.Bytes()
}

func (q *QPic) ToRGBAImage() *rgba.Image {
	return rgba.ToImage(q.Width, q.Height, q.Pixels)
}

func Parse(data []byte) (*QPic, error) {
	if len(data) < headerSize {
		return nil, ErrInvalidSize
	}

	buf := buffer.New(buffer.WithData(data))

	width, err := buf.GetUint32()
	if err != nil {
		return nil, err
	}

	height, err := buf.GetUint32()
	if err != nil {
		return nil, err
	}

	w, h := int(width), int(height)
	expectedSize := w*h + headerSize
	if len(data) != expectedSize {
		return nil, fmt.Errorf("expected %d bytes, got %d", expectedSize, len(data))
	}

	pixels, err := buf.GetBytes(w * h)
	if err != nil {
		return nil, err
	}

	return &QPic{
		Width:  w,
		Height: h,
		Pixels: pixels,
	}, nil
}

func FromPNG(data []byte) (*QPic, error) {
	rgba, err := rgba.FromPNG(data)
	if err != nil {
		return nil, err
	}

	return &QPic{
		Width:  rgba.Width,
		Height: rgba.Height,
		Pixels: rgba.Pixels,
	}, nil
}
