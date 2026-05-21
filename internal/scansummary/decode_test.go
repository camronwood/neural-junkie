package scansummary

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/tiff"
)

func TestIsTIFF(t *testing.T) {
	if !IsTIFF([]byte{'I', 'I', 42, 0}) {
		t.Fatal("expected little-endian TIFF")
	}
	if IsTIFF([]byte("PNG")) {
		t.Fatal("PNG should not be TIFF")
	}
}

func TestDecodeWellTIFFToPNG_roundTrip(t *testing.T) {
	rect := image.Rect(0, 0, 4, 4)
	gray := image.NewGray16(rect)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			gray.SetGray16(x, y, color.Gray16{Y: uint16((x + y) * 1000)})
		}
	}
	var tiffBuf bytes.Buffer
	if err := tiff.Encode(&tiffBuf, gray, nil); err != nil {
		t.Fatalf("encode tiff: %v", err)
	}
	pngBytes, err := DecodeWellTIFFToPNG(tiffBuf.Bytes())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pngBytes) < 8 || pngBytes[0] != 0x89 {
		t.Fatalf("expected PNG magic, got %v", pngBytes[:4])
	}
}
