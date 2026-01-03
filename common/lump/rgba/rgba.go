package rgba

import (
	"bytes"
	"errors"
	"image"
	"image/png"

	"github.com/osm/quake/common/palette"
)

const (
	minAlphaThreshold = 128
	transparentIndex  = 255
)

var (
	ErrNoImageData = errors.New("no image data")
)

type Image struct {
	Width  int
	Height int
	Pixels []byte
}

func (rgba *Image) ToPNG() ([]byte, error) {
	if rgba.Pixels == nil {
		return nil, ErrNoImageData
	}

	img := image.NewRGBA(image.Rect(0, 0, rgba.Width, rgba.Height))
	copy(img.Pix, rgba.Pixels)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ToImage(width, height int, pixels []byte) *Image {
	rgba := &Image{
		Width:  width,
		Height: height,
		Pixels: make([]byte, width*height*4),
	}

	for i, paletteIndex := range pixels {
		offset := i * 4
		rgba.Pixels[offset] = palette.Palette[paletteIndex].R
		rgba.Pixels[offset+1] = palette.Palette[paletteIndex].G
		rgba.Pixels[offset+2] = palette.Palette[paletteIndex].B
		rgba.Pixels[offset+3] = calculateAlpha(int(paletteIndex))
	}

	return rgba
}

func FromPNG(data []byte) (*Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	r := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r.Set(x, y, img.At(x, y))
		}
	}

	width := bounds.Dx()
	height := bounds.Dy()
	indexed, err := toIndexed(width, height, r.Pix)
	if err != nil {
		return nil, err
	}

	return &Image{
		Width:  width,
		Height: height,
		Pixels: indexed,
	}, nil
}

func toIndexed(width, height int, pixels []byte) ([]byte, error) {
	indexed := make([]byte, width*height)

	for i := 0; i < width*height; i++ {
		offset := i * 4
		r := pixels[offset]
		g := pixels[offset+1]
		b := pixels[offset+2]
		a := pixels[offset+3]

		if a < minAlphaThreshold {
			indexed[i] = transparentIndex
		} else {
			indexed[i] = findNearestPaletteColor(r, g, b)
		}
	}

	return indexed, nil
}

func findNearestPaletteColor(r, g, b uint8) uint8 {
	var bestIndex uint8
	bestDistance := int32(0x7fffffff)

	for i := 0; i < transparentIndex; i++ {
		dr := int32(r) - int32(palette.Palette[i].R)
		dg := int32(g) - int32(palette.Palette[i].G)
		db := int32(b) - int32(palette.Palette[i].B)

		distance := dr*dr + dg*dg + db*db

		if distance < bestDistance {
			bestDistance = distance
			bestIndex = uint8(i)

			if distance == 0 {
				break
			}
		}
	}

	return bestIndex
}

func calculateAlpha(paletteIndex int) uint8 {
	if paletteIndex == transparentIndex {
		return 0
	}

	return 255
}
