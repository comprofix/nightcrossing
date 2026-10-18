package play

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// young is the sprite the toad's line is drawn with: its toadlets, in
// school caps on the school gates (DESIGN §6.7).
func (p *Play) young() string {
	if b := p.world.Level.Bonus; b != nil && b.Stragglers > 0 {
		return p.hero.SchoolYoung
	}
	return p.hero.Young
}

// drawStragglers draws the school gates' toadlets still waiting at their
// kerbs, facing up the field, the way home (DESIGN §6.7).
func (p *Play) drawStragglers(ctx *scene.Context, dst *ebiten.Image) {
	for _, s := range p.world.Stragglers {
		if !s.Waiting() {
			continue
		}
		pl := world.Player{Col: s.Col, Row: s.Row, X: s.Col * cellUnits, Y: s.Row * cellUnits,
			Facing: world.Up, Still: config.LandTicks}
		p.drawToad(ctx, dst, &pl, p.young(), 0)
	}
}

// cellUnits is one cell in SubPx units.
const cellUnits = config.Cell * config.SubPx
