package gfx

import (
	"bytes"
	"math"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomonobold"

	"github.com/comprofix/nightcrossing/internal/config"
)

// VectorFont is a Font drawn from an outline font, smooth at any size: Go
// Mono Bold, for the smooth art style. Go Mono's characters are all one
// width, so text still lays out in character cells, though the cells are
// about half as wide as they are tall.
type VectorFont struct {
	face        *text.GoTextFace
	cell, line  int
	shadowShift int
}

// goMono is Go Mono Bold, read once.
var goMono = func() *text.GoTextFaceSource {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(gomonobold.TTF))
	if err != nil {
		panic("gfx: Go Mono Bold: " + err.Error())
	}
	return src
}()

// NewVectorFont makes Go Mono Bold at size pixels to the em. Each cell is
// a character's advance rounded to whole pixels (0.6 of the size, so a size
// that is a multiple of 5 fits its cells exactly), and the line height is
// the font's ascent and descent rounded up (about 1.16 of the size).
func NewVectorFont(size int) *VectorFont {
	face := &text.GoTextFace{Source: goMono, Size: float64(size)}
	m := face.Metrics()
	return &VectorFont{
		face:        face,
		cell:        int(math.Round(text.Advance("M", face))),
		line:        int(math.Ceil(m.HAscent + m.HDescent)),
		shadowShift: max(1, size/16),
	}
}

// CellWidth is how far each character moves the next one on, in pixels.
func (f *VectorFont) CellWidth() int { return f.cell }

// LineHeight is the height of one line, in pixels.
func (f *VectorFont) LineHeight() int { return f.line }

// Width is the width of s in pixels: one cell for each character.
func (f *VectorFont) Width(s string) int { return utf8.RuneCountInString(s) * f.cell }

// Draw renders s with the top of its line at y, in palette colour c. Each
// character is placed in its own cell, so text lines up with the cells
// exactly.
func (f *VectorFont) Draw(dst *ebiten.Image, s string, x, y, c int) {
	op := &text.DrawOptions{}
	op.ColorScale.ScaleWithColor(RGBA(c))
	for _, r := range s {
		if r != ' ' {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(x), float64(y))
			text.Draw(dst, string(r), f.face, op)
		}
		x += f.cell
	}
}

// DrawShadowed renders s with a drop shadow in palette colour shadow,
// offset down and right by about a sixteenth of the font's size.
func (f *VectorFont) DrawShadowed(dst *ebiten.Image, s string, x, y, c, shadow int) {
	d := f.shadowShift
	f.Draw(dst, s, x+d, y+d, shadow)
	f.Draw(dst, s, x, y, c)
}

// CentreX is the x that centres s on the screen.
func (f *VectorFont) CentreX(s string) int {
	return (config.Project.ScreenWidth - f.Width(s)) / 2
}
