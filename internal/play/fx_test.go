package play

import (
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// TestHopFrames checks every tick of a hop has a frame of the arc, and it
// runs crouch, stretch, leap, tuck (DESIGN §3.3).
func TestHopFrames(t *testing.T) {
	order := []int{frameCrouch, frameStretch, frameLeap, frameTuck}
	at := 0
	for i, f := range hopFrames {
		for at < len(order) && order[at] != f {
			at++
		}
		if at == len(order) {
			t.Fatalf("tick %d: frame %d out of order in %v", i+1, f, hopFrames)
		}
	}
	if hopFrames[0] != frameCrouch || hopFrames[len(hopFrames)-1] != frameTuck {
		t.Errorf("a hop should start crouched and end tucked: %v", hopFrames)
	}
}

// TestWheelsTurnAsTheyGo checks a vehicle's wheel frame changes every few
// pixels it moves, and only then.
func TestWheelsTurnAsTheyGo(t *testing.T) {
	step := wheelStepPx * config.SubPx
	e := world.Entity{U: 0}
	f0 := wheelFrame(e)
	e.U = step - 1
	if wheelFrame(e) != f0 {
		t.Error("the wheels turned before the vehicle went a step")
	}
	e.U = step
	if wheelFrame(e) == f0 {
		t.Error("the wheels did not turn after a step")
	}
}

// TestEffectsFollowEvents checks events start particles and a nook's glow,
// and that both run down.
func TestEffectsFollowEvents(t *testing.T) {
	h := scenetest.New(t, 5)
	h.Ctx.Session = session.New(config.Modes[0], config.StartLives)
	lv := config.Level{Number: 1, TimerTicks: config.TimerTicks(1)}
	lv.Lanes[config.GoalRow].Kind = config.Bank
	p := newPlay(h.Ctx, config.StrollLevels, lv)

	p.effects(world.Splashed)
	if p.parts.Len() == 0 {
		t.Fatal("a splash made no particles")
	}
	p.world.Player.Col = config.Nooks[2]
	p.effects(world.Goal)
	if p.nookGlow[2] != goalGlowTick {
		t.Fatalf("nook glow %v after a goal in nook 2", p.nookGlow)
	}
	for range 2 * goalGlowTick {
		p.updateEffects()
	}
	if p.parts.Len() != 0 || p.nookGlow[2] != 0 {
		t.Errorf("after %d ticks: %d particles, nook glow %v", 2*goalGlowTick, p.parts.Len(), p.nookGlow)
	}
}

// TestFlickerStaysInRange checks lights flicker between 85% and full, the
// same way each time.
func TestFlickerStaysInRange(t *testing.T) {
	seen := map[float64]bool{}
	for tick := range int64(400) {
		k := flicker(tick, 9, 2)
		if k < 0.85 || k > 1 {
			t.Fatalf("tick %d: flicker %v", tick, k)
		}
		if k != flicker(tick, 9, 2) {
			t.Fatal("flicker differs for the same tick")
		}
		seen[k] = true
	}
	if len(seen) < 10 {
		t.Errorf("only %d flicker levels in 400 ticks", len(seen))
	}
}
