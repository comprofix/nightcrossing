package play

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/scene"
)

// The play scene, the title's demo, GAME OVER, MASTERED and the ending
// draw in either look (DESIGN §3.6): the game is in the pixel look's
// coordinates, and the smooth look shows them twice the size, its art
// drawn in code at the screen's size (see smooth.go), its text in the
// shown look's font.

// Native implements scene.Native.
func (*Play) Native()  {}
func (*Demo) Native()  {}
func (*Ended) Native() {}

// gamePreset is the look the game's coordinates are in: the pixel look of
// two, or the only one.
func gamePreset(ctx *scene.Context) config.Preset {
	if px := ctx.Looks.Pixel(); px != nil {
		return *px
	}
	return ctx.Preset
}

// scale is how many screen pixels each of the game's is across in the look
// shown: 1 in the pixel look, 2 in the smooth one.
func (p *Play) scale(ctx *scene.Context) int {
	return max(ctx.Preset.ScreenWidth/p.size.X, 1)
}

// inPixels runs draw on an image in the game's coordinates and shows what
// it drew on dst, enlarged k times with sharp pixels: the art that has no
// smooth form yet. At k 1 it draws on dst itself.
func (p *Play) inPixels(dst *ebiten.Image, k int, draw func(img *ebiten.Image)) {
	if k == 1 {
		draw(dst)
		return
	}
	if p.art == nil {
		p.art = ebiten.NewImage(p.size.X, p.size.Y)
	}
	p.art.Clear()
	draw(p.art)
	enlarge(dst, p.art, k)
}

// enlarge draws src on dst k times its size, with sharp pixels.
func enlarge(dst, src *ebiten.Image, k int) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(float64(k), float64(k))
	dst.DrawImage(src, op)
}

// drawIcon draws frame i of sprite name with its top-left corner at x, y
// in the game's coordinates, k times its size: in its smooth art in the
// smooth look, as the HUD's icons are.
func (p *Play) drawIcon(ctx *scene.Context, dst *ebiten.Image, name string, i int, x, y float64, k int) {
	sp := ctx.Sheet(Sheet).Sprite(name)
	if k > 1 {
		if img := p.artStamp(name, i, k, sp.W, sp.H); img != nil {
			lay(dst, img, k, x, y, 1)
			return
		}
	}
	drawSprite(dst, sp, i, x, y, k)
}

// drawSprite draws frame i of sp with its top-left corner at x, y in the
// game's coordinates, k times its size with sharp pixels.
func drawSprite(dst *ebiten.Image, sp *gfx.Sprite, i int, x, y float64, k int) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(x, y)
	op.GeoM.Scale(float64(k), float64(k))
	dst.DrawImage(sp.Frame(i), op)
}
