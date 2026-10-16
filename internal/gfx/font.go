package gfx

import (
	"fmt"
	"image"
	"image/color"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/bitfont"
	"github.com/comprofix/gameskeleton/internal/config"
)

// Font draws text in any palette colour.
type Font struct {
	*bitfont.Font
	atlas *ebiten.Image
	index map[rune]int
}

// Fonts are the game's two fonts.
type Fonts struct {
	Small *Font // 8×8
	Large *Font // 16×16, made from Small with Scale2x
}

// LoadFonts reads the 8×8 font from fsys and makes the large font from it.
func LoadFonts(fsys fs.FS, path string) (Fonts, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return Fonts{}, err
	}
	defer f.Close()
	small, err := bitfont.Parse(f)
	if err != nil {
		return Fonts{}, fmt.Errorf("%s: %w", path, err)
	}
	return Fonts{Small: NewFont(small), Large: NewFont(bitfont.Scale2x(small))}, nil
}

// NewFont builds a glyph atlas for bf. Ink is stored white and tinted when
// drawn.
func NewFont(bf *bitfont.Font) *Font {
	runes := bf.Runes()
	n := bf.Size
	img := image.NewRGBA(image.Rect(0, 0, n*len(runes), n))
	f := &Font{Font: bf, index: make(map[rune]int, len(runes))}
	for i, r := range runes {
		bf.Draw(img.SubImage(image.Rect(i*n, 0, (i+1)*n, n)).(*image.RGBA), string(r), i*n, 0, color.RGBA{0xff, 0xff, 0xff, 0xff})
		f.index[r] = i
	}
	f.atlas = ebiten.NewImageFromImage(img)
	return f
}

// Draw renders s at (x, y) in palette colour c. Characters missing from the
// font are drawn as '?'.
func (f *Font) Draw(dst *ebiten.Image, s string, x, y, c int) {
	n := f.Size
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleWithColor(RGBA(c))
	for _, r := range s {
		i, ok := f.index[r]
		if !ok {
			i, ok = f.index['?']
		}
		if ok && r != ' ' {
			src := f.atlas.SubImage(image.Rect(i*n, 0, (i+1)*n, n)).(*ebiten.Image)
			op.GeoM.Reset()
			op.GeoM.Translate(float64(x), float64(y))
			dst.DrawImage(src, op)
		}
		x += n
	}
}

// DrawShadowed renders s with a drop shadow in palette colour shadow, offset
// down and right by one pixel per 8 pixels of font size.
func (f *Font) DrawShadowed(dst *ebiten.Image, s string, x, y, c, shadow int) {
	d := max(1, f.Size/8)
	f.Draw(dst, s, x+d, y+d, shadow)
	f.Draw(dst, s, x, y, c)
}

// CentreX is the x that centres s on the screen.
func (f *Font) CentreX(s string) int {
	return (config.Project.ScreenWidth - f.Width(s)) / 2
}
