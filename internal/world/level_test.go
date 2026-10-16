package world

import (
	"math/rand/v2"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// lastLevel is the first level played at the speed ceiling, plus one
// more round of the loop: past it, nothing gets faster.
func lastLevel(m config.ModeLevels) int {
	n := len(m.Levels) + 1
	for {
		if _, s := m.Def(n); s >= m.CeilingPct {
			return n + config.LoopLevels
		}
		n++
	}
}

// TestEveryLevelCanBeCleared plays every level of Stroll, through the
// loops to the speed ceiling, with the careful player from the river test:
// across the road when the lane ahead is clear, across the river on
// platforms that will hold it, and up into empty nooks. It has as many
// lives as it needs, but must fill all five nooks within a few minutes of
// play, from several seeds. The crossing check in config proves each
// level can be crossed; this shows the whole level plays through.
func TestEveryLevelCanBeCleared(t *testing.T) {
	const limit = 5 * 60 * config.TPS
	for n := 1; n <= lastLevel(config.StrollLevels); n++ {
		lv := config.StrollLevels.Level(n)
		deaths := 0
		for seed := uint64(1); seed <= 3; seed++ {
			w := New(session.New(config.Modes[0], config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
			for !w.Cleared {
				if w.Tick > limit {
					t.Fatalf("level %d seed %d: %d nooks filled after %d ticks; hero in row %d at %d, filled %v, dying %d home %d", n, seed, w.Stats.Goals, limit, w.Player.Row, w.Player.X/config.SubPx, w.Filled, w.Dying, w.Home)
				}
				w.Update(carefulMove(w))
				if w.Dying == config.DeathTicks {
					deaths++
					w.Session.Lives = config.StartLives // as many as it needs
				}
			}
		}
		_, speed := config.StrollLevels.Def(n)
		t.Logf("level %2d (%3d%% speed): cleared from 3 seeds with %d deaths", n, speed, deaths)
	}
}
