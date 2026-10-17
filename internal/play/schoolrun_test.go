package play

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// schoolRunPlay is a School Run game with a line of n toadlets on an
// all-safe level with the far bank at the top, faded in and ready.
func schoolRunPlay(t *testing.T, n int, lanes func(*config.Level)) (*scenetest.Harness, *Play) {
	h := scenetest.New(t, 5)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.SchoolRun, config.StartLives)
	lv := config.Level{Number: 1, TimerTicks: config.TimerTicks(1), Toadlets: n}
	lv.Lanes[config.GoalRow].Kind = config.Bank
	if lanes != nil {
		lanes(&lv)
	}
	p := newPlay(h.Ctx, config.SchoolRun.Levels, lv)
	h.Start(p)
	h.Ticks(scene.FadeTicks + 1)
	return h, p
}

// hop presses up and runs the hop out.
func hop(h *scenetest.Harness) {
	h.Run(scenetest.Hold(1, ebiten.KeyArrowUp), scenetest.Hold(config.HopTicks+1))
}

// TestAToadletCriesFirst: a toadlet's death cries at once, and the death's
// own sound waits cryLeadTicks to play after it (D56).
func TestAToadletCriesFirst(t *testing.T) {
	const road = config.StartRow - 1
	h, p := schoolRunPlay(t, 1, func(lv *config.Level) {
		lv.Lanes[road] = config.Lane{Kind: config.Road, Dir: 1, Speed: config.SubPx,
			Pattern: []config.Spawn{{Entity: config.CarRed, Gap: 600}}}
	})
	w := p.World()
	hop(h) // onto the road
	hop(h) // off it, with the toadlet hopping on behind
	// A car comes along the road, onto the toadlet.
	w.Lanes[road].Entities = []world.Entity{{Kind: config.CarRed,
		U: config.StartCol*config.Cell*config.SubPx - config.CarRed.Length*config.SubPx + world.Buffer}}
	for range config.TPS {
		if h.Ticks(1); w.Dying > 0 {
			break
		}
	}
	if w.Dying == 0 || w.Fallen() == &w.Player {
		t.Fatalf("dying %d, fallen %v: want the toadlet hit", w.Dying, w.Fallen())
	}
	if len(p.later) != 1 || p.later[0].effect != Crash || p.later[0].at != p.ticks+cryLeadTicks {
		t.Fatalf("waiting %+v at tick %d: want the crash %d ticks after the cry", p.later, p.ticks, cryLeadTicks)
	}
	h.Ticks(cryLeadTicks)
	if len(p.later) != 0 {
		t.Errorf("still waiting %+v after the lead", p.later)
	}
}

// TestToadletsPeepHome: a nook filled with the line peeps for each toadlet
// after the home run, peepGapTicks apart (D56).
func TestToadletsPeepHome(t *testing.T) {
	h, p := schoolRunPlay(t, 2, nil)
	w := p.World()
	for range config.StartRow - config.GoalRow - 1 {
		hop(h)
	}
	run := ticks(Home.Part.Notes)
	h.Run(scenetest.Hold(1, ebiten.KeyArrowUp))
	for w.Home == 0 {
		h.Ticks(1)
	}
	if len(p.later) != 2 {
		t.Fatalf("waiting %+v: want a peep for each toadlet", p.later)
	}
	home := p.ticks
	for i, s := range p.later {
		if s.effect != Toad.Peep || s.at != home+run+i*peepGapTicks {
			t.Errorf("sound %d: %s at %d, want the peep at %d", i, s.effect.Name, s.at, home+run+i*peepGapTicks)
		}
	}
	h.Ticks(run + 2*peepGapTicks)
	if len(p.later) != 0 {
		t.Errorf("still waiting %+v", p.later)
	}
}
