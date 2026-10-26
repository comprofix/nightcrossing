package play

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/scene"
)

// The play scene, the title's demo and GAME OVER draw in either look
// (DESIGN §3.6): the game is in the pixel look's coordinates, and the
// smooth look shows them twice the size. The field's art is still the pixel
// sprites, drawn at their own size and enlarged with sharp pixels; the
// light, glow, rain, wind and particles are drawn at the screen's size, and
// the status bar and hint strip in the shown look's font.

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

// drawSprite draws frame i of sp with its top-left corner at x, y in the
// game's coordinates, k times its size with sharp pixels.
func drawSprite(dst *ebiten.Image, sp *gfx.Sprite, i int, x, y float64, k int) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(x, y)
	op.GeoM.Scale(float64(k), float64(k))
	dst.DrawImage(sp.Frame(i), op)
}
