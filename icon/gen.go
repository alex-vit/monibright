package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
)

var (
	colorLow  = color.NRGBA{R: 0xB8, G: 0x73, B: 0x33, A: 0xFF} // deep amber #B87333
	colorMid  = color.NRGBA{R: 0xDC, G: 0xA5, B: 0x1A, A: 0xFF} // warm yellow #DCA51A
	colorHigh = color.NRGBA{R: 0xFF, G: 0xD7, B: 0x00, A: 0xFF} // bright gold #FFD700
)

// Generate returns ICO bytes (16+32 px) for a brightness level 0-100.
func Generate(level int) []byte {
	if level < 0 {
		level = 0
	} else if level > 100 {
		level = 100
	}

	t := float64(level) / 100.0
	c := sunColor(t)

	sizes := []int{16, 32}
	var pngs [][]byte
	for _, size := range sizes {
		var buf bytes.Buffer
		png.Encode(&buf, eclipseImage(size, t, c))
		pngs = append(pngs, buf.Bytes())
	}
	return buildICO(sizes, pngs)
}

// eclipseImage draws a sun circle partially eclipsed by a moon circle.
// The moon slides from fully overlapping (t=0, full eclipse) to fully
// off-screen (t=1, full sun). No anti-aliasing — every pixel is either
// fully opaque or fully transparent.
func eclipseImage(size int, t float64, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	center := float64(size) / 2
	sunR := center - 0.5
	moonR := sunR

	// Moon displacement: 0 at t=0 (full eclipse) to beyond sun edge at t=1.
	moonCx := center + t*(2*sunR+1)

	for y := range size {
		for x := range size {
			px := float64(x) + 0.5
			py := float64(y) + 0.5

			sunDist := math.Hypot(px-center, py-center)
			moonDist := math.Hypot(px-moonCx, py-center)

			if sunDist <= sunR && moonDist > moonR {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	return img
}

func sunColor(t float64) color.NRGBA {
	if t <= 0.5 {
		return lerpColor(colorLow, colorMid, t*2)
	}
	return lerpColor(colorMid, colorHigh, (t-0.5)*2)
}

func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: 0xFF,
	}
}

// buildICO assembles an ICO file from PNG-encoded images.
func buildICO(sizes []int, pngs [][]byte) []byte {
	n := len(sizes)
	dataOffset := 6 + n*16

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, [3]uint16{0, 1, uint16(n)})

	offset := uint32(dataOffset)
	for i, size := range sizes {
		w := uint8(size)
		if size >= 256 {
			w = 0
		}
		buf.Write([]byte{w, w, 0, 0})
		binary.Write(&buf, binary.LittleEndian, uint16(1))
		binary.Write(&buf, binary.LittleEndian, uint16(32))
		binary.Write(&buf, binary.LittleEndian, uint32(len(pngs[i])))
		binary.Write(&buf, binary.LittleEndian, offset)
		offset += uint32(len(pngs[i]))
	}

	for _, p := range pngs {
		buf.Write(p)
	}
	return buf.Bytes()
}
