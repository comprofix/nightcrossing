package play

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/scene"
)

// The hop combo's readout in the status bar, between the lives and the
// timer (DESIGN §6.2, D38), and its pop-ups.
const (
	hudComboX   = 408
	comboBarW   = 32
	comboBarH   = 4
	popTicks    = 40 // a pop-up shows this long
	popRiseTick = 3  // and rises a pixel this often
)

// pop is points popping up over the toad as the combo scores them.
type pop struct {
	x, y  int
	text  string
	ticks int
}

// popGained adds a pop-up for the points the world popped this tick, if
// any: what the combo multiplied, or a firefly's at full lives (D38, D57).
func (p *Play) popGained() {
	if g := p.world.Gained; g > 0 {
		p.popUp(fmt.Sprintf("+%d", g), 0)
	}
}

// oneUpLiftPx is how far above a points pop-up a 1UP's starts, so the two
// can show together.
const oneUpLiftPx = 10

// popUp adds a pop-up of text over the toad, lift pixels higher than
// usual.
func (p *Play) popUp(text string, lift int) {
	x, y := p.heroCentre()
	p.pops = append(p.pops, pop{x: int(x), y: int(y) - config.Cell/2 - lift, text: text})
}

// updatePops ages the pop-ups and drops the old ones.
func (p *Play) updatePops() {
	kept := p.pops[:0]
	for _, pp := range p.pops {
		if pp.ticks++; pp.ticks < popTicks {
			kept = append(kept, pp)
		}
	}
	p.pops = kept
}

// drawPops draws the pop-ups, rising as they age, centred on where they
// started, k times the size.
func (p *Play) drawPops(ctx *scene.Context, dst *ebiten.Image, k int) {
	sc, f := ctx.Scheme, ctx.Fonts
	for _, pp := range p.pops {
		x := k*pp.x - f.Small.Width(pp.text)/2
		f.Small.DrawShadowed(dst, pp.text, x, k*(pp.y-pp.ticks/popRiseTick)-f.Small.LineHeight(), sc.Highlight, sc.Shadow)
	}
}

// drawCombo draws the combo's multiplier, from x2, with a bar showing how
// long is left to keep it going, k times the size.
func (p *Play) drawCombo(ctx *scene.Context, dst *ebiten.Image, ty, k int) {
	c := p.world.Combo
	if c.Mult() < 2 {
		return
	}
	sc, f := ctx.Scheme, ctx.Fonts
	label := fmt.Sprintf("x%d", c.Mult())
	f.Small.DrawShadowed(dst, label, k*hudComboX, ty, sc.Highlight, sc.Shadow)
	kf := float32(k)
	bx := kf*hudComboX + float32(f.Small.Width(label)) + kf*hudLifeGap*2
	by := kf * (config.HUDHeight - comboBarH) / 2
	vector.FillRect(dst, bx, by, kf*comboBarW, kf*comboBarH, gfx.RGBA(sc.Panel), false)
	vector.FillRect(dst, bx, by, kf*float32(comboBarW*c.Left/config.ComboTicks), kf*comboBarH,
		gfx.RGBA(sc.Highlight), false)
}
