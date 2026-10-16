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

// popGained adds a pop-up for the points the combo multiplied this tick,
// if any.
func (p *Play) popGained() {
	if g := p.world.Gained; g > 0 {
		x, y := p.heroCentre()
		p.pops = append(p.pops, pop{x: int(x), y: int(y) - config.Cell/2, text: fmt.Sprintf("+%d", g)})
	}
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
// started.
func (p *Play) drawPops(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	for _, pp := range p.pops {
		x := pp.x - len(pp.text)*f.Small.Size/2
		f.Small.DrawShadowed(dst, pp.text, x, pp.y-f.Small.Size-pp.ticks/popRiseTick, sc.Highlight, sc.Shadow)
	}
}

// drawCombo draws the combo's multiplier, from x2, with a bar showing how
// long is left to keep it going.
func (p *Play) drawCombo(ctx *scene.Context, dst *ebiten.Image, ty int) {
	c := p.world.Combo
	if c.Mult() < 2 {
		return
	}
	sc, f := ctx.Scheme, ctx.Fonts
	label := fmt.Sprintf("x%d", c.Mult())
	f.Small.DrawShadowed(dst, label, hudComboX, ty, sc.Highlight, sc.Shadow)
	bx := hudComboX + len(label)*f.Small.Size + hudLifeGap*2
	by := (config.HUDHeight - comboBarH) / 2
	vector.FillRect(dst, float32(bx), float32(by), comboBarW, comboBarH, gfx.RGBA(sc.Panel), false)
	vector.FillRect(dst, float32(bx), float32(by), float32(comboBarW*c.Left/config.ComboTicks), comboBarH,
		gfx.RGBA(sc.Highlight), false)
}
