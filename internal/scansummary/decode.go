package scansummary

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"

	"golang.org/x/image/tiff"
)

// IsTIFF reports whether data begins with a little- or big-endian TIFF signature.
func IsTIFF(data []byte) bool {
	return len(data) >= 4 &&
		((data[0] == 'I' && data[1] == 'I' && data[2] == 42 && data[3] == 0) ||
			(data[0] == 'M' && data[1] == 'M' && data[2] == 0 && data[3] == 42))
}

// DecodeWellTIFFToPNG decodes a well TIFF (often 32-bit grayscale) to 8-bit PNG bytes.
func DecodeWellTIFFToPNG(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty TIFF data")
	}
	img, err := tiff.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode TIFF: %w", err)
	}
	gray8 := toGray8(img)
	var buf bytes.Buffer
	if err := png.Encode(&buf, gray8); err != nil {
		return nil, fmt.Errorf("encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func toGray8(src image.Image) *image.Gray {
	b := src.Bounds()
	out := image.NewGray(b)
	minV := math.MaxFloat64
	maxV := -math.MaxFloat64

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, g, _, a := src.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			v := float64(g) / 65535.0
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
	}
	if maxV < minV {
		return out
	}
	if maxV <= minV {
		maxV = minV + 1
	}
	scale := 255.0 / (maxV - minV)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, g, _, a := src.At(x, y).RGBA()
			if a == 0 {
				out.SetGray(x, y, color.Gray{Y: 0})
				continue
			}
			v := float64(g) / 65535.0
			n := uint8(math.Round((v - minV) * scale))
			out.SetGray(x, y, color.Gray{Y: n})
		}
	}
	return out
}
