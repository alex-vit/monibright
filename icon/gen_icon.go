//go:build ignore

// Generates brightness.ico (yellow circle at 16, 32, 48 px).
// Invoked by: go generate ./internal/icon
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

var yellow = color.NRGBA{R: 0xFF, G: 0xD7, B: 0x00, A: 0xFF}

func main() {
	sizes := []int{16, 32, 48}

	var pngs [][]byte
	for _, size := range sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, circle(size, yellow)); err != nil {
			fmt.Fprintf(os.Stderr, "png encode %d: %v\n", size, err)
			os.Exit(1)
		}
		pngs = append(pngs, buf.Bytes())
	}

	if err := os.WriteFile("brightness.ico", ico(sizes, pngs), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote brightness.ico")
}

// circle draws an anti-aliased filled circle centered in a size×size image.
func circle(size int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2
	radius := center - 0.5 // half-pixel inset so edges don't clip
	for y := range size {
		for x := range size {
			dx := float64(x) + 0.5 - center
			dy := float64(y) + 0.5 - center
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= radius-0.5 {
				img.SetNRGBA(x, y, c)
			} else if dist <= radius+0.5 {
				alpha := uint8(float64(c.A) * (radius + 0.5 - dist))
				img.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: alpha})
			}
		}
	}
	return img
}

// ico builds an ICO file from PNG-encoded images.
func ico(sizes []int, pngs [][]byte) []byte {
	n := len(sizes)
	dataOffset := 6 + n*16 // header + directory entries

	var buf bytes.Buffer
	// Header: reserved, type (1=ICO), count.
	binary.Write(&buf, binary.LittleEndian, [3]uint16{0, 1, uint16(n)})

	// Directory entries.
	offset := uint32(dataOffset)
	for i, size := range sizes {
		w := uint8(size)
		if size >= 256 {
			w = 0
		}
		buf.Write([]byte{w, w, 0, 0})                                 // width, height, palette, reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))            // color planes
		binary.Write(&buf, binary.LittleEndian, uint16(32))           // bits per pixel
		binary.Write(&buf, binary.LittleEndian, uint32(len(pngs[i]))) // data size
		binary.Write(&buf, binary.LittleEndian, offset)               // data offset
		offset += uint32(len(pngs[i]))
	}

	// Image data.
	for _, p := range pngs {
		buf.Write(p)
	}
	return buf.Bytes()
}
