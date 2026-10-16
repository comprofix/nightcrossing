package world

import (
	"math/rand/v2"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

func TestCloneRunsOnItsOwn(t *testing.T) {
	w := New(session.New(config.Stroll, config.StartLives), rich, rand.New(rand.NewPCG(1, 1)))
	tick, u, lives := w.Tick, w.Lanes[1].Entities[0].U, w.Session.Lives
	c := w.Clone(rand.New(rand.NewPCG(2, 2)))
	c.Session.Lives--
	for range 100 {
		c.Update(Up)
	}
	if w.Tick != tick || w.Lanes[1].Entities[0].U != u || w.Session.Lives != lives {
		t.Errorf("running the clone changed the world")
	}
}

// TestAutopilotClearsTheDemoLevels plays the title's demo levels from the
// demo's seed: the autopilot must clear each one without losing a life.
func TestAutopilotClearsTheDemoLevels(t *testing.T) {
	for n := 1; n <= config.DemoLevels; n++ {
		s := session.New(config.Stroll, config.StartLives)
		w := New(s, config.StrollLevels.Level(n), rand.New(rand.NewPCG(config.DemoSeed, uint64(n))))
		var a Autopilot
		for range 60 * config.TPS {
			if w.Update(a.Next(w)); w.Cleared || w.Over {
				break
			}
		}
		if !w.Cleared || s.Lives != config.StartLives {
			t.Errorf("level %d: cleared %v after %d s, %d lives lost", n, w.Cleared, w.Tick/config.TPS, config.StartLives-s.Lives)
		}
	}
}
