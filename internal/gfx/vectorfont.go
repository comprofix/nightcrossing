package gfx

import (
	"bytes"
	"math"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gomonobold"
)

// VectorFont is a Font drawn from an outline font, smooth at any size: Go
// Mono Bold, for the smooth art style. Go Mono's characters are all one
// width, so text still lays out in character cells, though the cells are
// about half as wide as they are tall.
type VectorFont struct {
	face        *text.GoTextFace
	cell, line  int
	shadowShift int
	capTop      float64 // from the top of a line to the top of the capitals
	base        float64 // from the top of a line to the baseline
}

// marks are the characters Go Mono lacks that the bitmap font has, drawn
// as shapes in a capital's box instead: the tick and the lock.
var marks = map[rune]func(dst *ebiten.Image, x, y, w, h, stroke float32, op *vector.DrawPathOptions){
	'✓': drawTick,
	'🔒': drawLock,
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
		capTop:      m.HAscent - m.CapHeight,
		base:        m.HAscent,
	}
}

// SmoothFonts are the smooth art style's two fonts for a screen
// screenHeight pixels tall: the small one has a size of a thirty-sixth of
// the height (20 at 720, cells 12 wide and 24 tall), the large one twice
// that.
func SmoothFonts(screenHeight int) Fonts {
	small := screenHeight / 36
	return Fonts{Small: NewVectorFont(small), Large: NewVectorFont(2 * small)}
}

// CellWidth is how far each character moves the next one on, in pixels.
func (f *VectorFont) CellWidth() int { return f.cell }

// LineHeight is the height of one line, in pixels.
func (f *VectorFont) LineHeight() int { return f.line }

// Width is the width of s in pixels: one cell for each character.
func (f *VectorFont) Width(s string) int { return utf8.RuneCountInString(s) * f.cell }

// Draw renders s with the top of its line at y, in palette colour c. Each
// character is placed in its own cell, so text lines up with the cells
// exactly; the marks are drawn as shapes.
func (f *VectorFont) Draw(dst *ebiten.Image, s string, x, y, c int) {
	op := &text.DrawOptions{}
	op.ColorScale.ScaleWithColor(RGBA(c))
	for _, r := range s {
		if mark, ok := marks[r]; ok {
			f.drawMark(dst, mark, x, y, c)
		} else if r != ' ' {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(x), float64(y))
			text.Draw(dst, string(r), f.face, op)
		}
		x += f.cell
	}
}

// drawMark draws a mark in the cell at x, y: as tall as a capital and as
// wide as the cell less a tenth each side, with lines as thick as Go Mono
// Bold's strokes, about a seventh of the size.
func (f *VectorFont) drawMark(dst *ebiten.Image, mark func(*ebiten.Image, float32, float32, float32, float32, float32, *vector.DrawPathOptions), x, y, c int) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(RGBA(c))
	pad := float32(f.cell) / 10
	top := float32(y) + float32(f.capTop)
	mark(dst, float32(x)+pad, top, float32(f.cell)-2*pad, float32(f.base-f.capTop), float32(f.face.Size/7), op)
}

// drawTick draws a tick in the box at x, y, w wide and h tall: a short
// stroke down to the bottom and a long one up to the top right, as the
// bitmap font's.
func drawTick(dst *ebiten.Image, x, y, w, h, stroke float32, op *vector.DrawPathOptions) {
	var p vector.Path
	p.MoveTo(x+stroke/2, y+h*0.55)
	p.LineTo(x+w*0.38, y+h-stroke/2)
	p.LineTo(x+w-stroke/2, y+stroke/2)
	vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: stroke, LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound}, op)
}

// drawLock draws a padlock in the box at x, y, w wide and h tall: a
// shackle over the top half and a body over the bottom, with a keyhole
// cut through it, as the bitmap font's.
func drawLock(dst *ebiten.Image, x, y, w, h, stroke float32, op *vector.DrawPathOptions) {
	bodyTop := y + h*0.45
	r := w/2 - stroke // the shackle's legs stand inside the body's sides
	var s vector.Path
	s.MoveTo(x+w/2-r, bodyTop)
	s.LineTo(x+w/2-r, y+stroke/2+r)
	s.Arc(x+w/2, y+stroke/2+r, r, math.Pi, 0, vector.Clockwise)
	s.LineTo(x+w/2+r, bodyTop)
	vector.StrokePath(dst, &s, &vector.StrokeOptions{Width: stroke}, op)

	var b vector.Path
	b.MoveTo(x, bodyTop)
	b.LineTo(x+w, bodyTop)
	b.LineTo(x+w, y+h)
	b.LineTo(x, y+h)
	b.Close()
	kw, kt, kb := stroke*0.8, bodyTop+(y+h-bodyTop)*0.25, y+h-(y+h-bodyTop)*0.3
	b.MoveTo(x+w/2-kw/2, kt)
	b.LineTo(x+w/2+kw/2, kt)
	b.LineTo(x+w/2+kw/2, kb)
	b.LineTo(x+w/2-kw/2, kb)
	b.Close()
	vector.FillPath(dst, &b, &vector.FillOptions{FillRule: vector.FillRuleEvenOdd}, op)
}

// DrawShadowed renders s with a drop shadow in palette colour shadow,
// offset down and right by about a sixteenth of the font's size.
func (f *VectorFont) DrawShadowed(dst *ebiten.Image, s string, x, y, c, shadow int) {
	d := f.shadowShift
	f.Draw(dst, s, x+d, y+d, shadow)
	f.Draw(dst, s, x, y, c)
}

// CentreX is the x that centres s across dst.
func (f *VectorFont) CentreX(dst *ebiten.Image, s string) int {
	return (dst.Bounds().Dx() - f.Width(s)) / 2
}
