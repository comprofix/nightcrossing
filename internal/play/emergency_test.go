package play

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// TestEmergencyReplay plays Rush Hour's level 8 from a fixed seed with the
// autopilot at the keys, and logs each emergency vehicle as it comes and
// goes: its warning, with its lane and vehicle, its run, its going, the
// queues moving off, and each death and nook filled (DESIGN §6.9, D72).
// A change to its rules or timings shows up here; if it is intended, run
// the test with -update.
func TestEmergencyReplay(t *testing.T) {
	emergencyReplay(t, config.RushHour, 8, 7, "emergency")
}

// TestEmergencyEscortReplay is TestEmergencyReplay in School Run, on a
// level with mixed traffic, so the toadlet goes through the queues and gets
// out of the emergency lane with the hero (D53, D72).
func TestEmergencyEscortReplay(t *testing.T) {
	emergencyReplay(t, config.SchoolRun, 6, 7, "emergency_escort")
}

// emergencyReplay plays mode's level n from seed with the autopilot, and
// compares its log of emergency vehicles with testdata/name.golden.
func emergencyReplay(t *testing.T, mode config.Mode, n int, seed uint64, name string) {
	h := scenetest.New(t, seed)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(mode, config.StartLives)
	m := mode.Levels
	p := newPlay(h.Ctx, m, m.Level(n))
	h.Start(p)
	h.Ticks(scene.FadeTicks)

	keys := map[world.Dir]ebiten.Key{world.Up: ebiten.KeyArrowUp, world.Down: ebiten.KeyArrowDown,
		world.Left: ebiten.KeyArrowLeft, world.Right: ebiten.KeyArrowRight}
	var log strings.Builder
	var a world.Autopilot
	w := p.World()
	goals, lives, phase := 0, h.Ctx.Session.Lives, w.Siren.Phase
	for range 150 * config.TPS {
		if key, ok := keys[a.Next(w)]; ok {
			h.Run(scenetest.Hold(1, key))
		} else {
			h.Ticks(1)
		}
		if w = p.World(); w.Over || w.Level.Number != n {
			break
		}
		if s := w.Siren; s.Phase != phase {
			switch s.Phase {
			case world.SirenWarning:
				fmt.Fprintf(&log, "tick %d: the %s coming down row %d\n", w.Tick, s.Responder().ID, s.Row)
			case world.SirenRun:
				fmt.Fprintf(&log, "tick %d: it comes through\n", w.Tick)
			case world.SirenPassed:
				fmt.Fprintf(&log, "tick %d: it has gone\n", w.Tick)
			case world.SirenReturn:
				fmt.Fprintf(&log, "tick %d: the queues move off\n", w.Tick)
			}
			phase = s.Phase
		}
		if w.Stats.Goals != goals {
			goals = w.Stats.Goals
			fmt.Fprintf(&log, "tick %d: nook filled, %d of 5\n", w.Tick, goals)
		}
		if l := h.Ctx.Session.Lives; l < lives {
			fmt.Fprintf(&log, "tick %d: died (%v) in row %d\n", w.Tick, w.Player.Death, w.Player.LaneRow())
		}
		lives = h.Ctx.Session.Lives
	}
	if s := log.String(); !strings.Contains(s, "comes through") {
		t.Errorf("no emergency vehicle came:\n%s", s)
	}
	scenetest.Golden(t, name, log.String()+summary(p, h.Ctx))
}
