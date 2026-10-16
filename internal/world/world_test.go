package world

import (
	"math/rand/v2"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
)

func TestNew(t *testing.T) {
	sk := config.Skills[1]
	w := New(config.Cols, config.Rows, sk, rand.New(rand.NewPCG(1, 1)))
	if w.Cols != 26 || w.Rows != 13 {
		t.Errorf("grid %d×%d, want 26×13", w.Cols, w.Rows)
	}
	if w.Skill.ID != sk.ID {
		t.Errorf("skill %q, want %q", w.Skill.ID, sk.ID)
	}
	if want := sk.TimerSecs * config.TPS; w.TicksLeft != want {
		t.Errorf("%d ticks on the clock, want %d", w.TicksLeft, want)
	}
}

func TestUpdateIsDeterministic(t *testing.T) {
	run := func() World {
		w := New(config.Cols, config.Rows, config.Skills[0], rand.New(rand.NewPCG(42, 42)))
		for range 600 {
			w.Update()
		}
		w.rng = nil // compared by value below; the generators are separate
		return *w
	}
	a, b := run(), run()
	if a != b {
		t.Errorf("two runs from one seed differ:\n%+v\n%+v", a, b)
	}
	if a.Tick != 600 {
		t.Errorf("tick %d after 600 updates", a.Tick)
	}
}
