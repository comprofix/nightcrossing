package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// comboWorld is a Last Train game on an all-safe level with the hop combo.
func comboWorld() *World {
	lv := testLevel()
	lv.Combo = true
	return New(session.New(config.LastTrain, config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
}

// hops hops the hero d n times, each landing before the next.
func hops(w *World, d Dir, n int) []Event {
	var all []Event
	for range n {
		all = append(all, run(w, d, config.HopTicks)...)
	}
	return all
}

func count(ev []Event, e Event) int {
	n := 0
	for _, x := range ev {
		if x == e {
			n++
		}
	}
	return n
}

func TestComboMult(t *testing.T) {
	for n, want := range []int{1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 4, 4} {
		if got := config.ComboMult(n); got != want {
			t.Errorf("%d hops: x%d, want x%d", n, got, want)
		}
	}
}

// TestComboBuildsTheMultiplier: x2 from the third hop, x3 from the sixth
// and x4 from the ninth, each rise giving out ComboUp (D38).
func TestComboBuildsTheMultiplier(t *testing.T) {
	w := comboWorld()
	ev := hops(w, Up, 9)
	if n := count(ev, ComboUp); n != 3 {
		t.Errorf("%d ComboUp events, want 3", n)
	}
	want := 10*2 + 20*3 + 30*3 + 40*1
	if w.Session.Score != want {
		t.Errorf("score %d after 9 hops, want %d", w.Session.Score, want)
	}
	if w.Stats.Hops != 9*config.PointsHop || w.Stats.Combo != want-9*config.PointsHop {
		t.Errorf("stats %+v: want hops %d and combo %d", w.Stats, 9*config.PointsHop, want-9*config.PointsHop)
	}
	if w.Combo.Mult() != 4 {
		t.Errorf("multiplier x%d, want x4", w.Combo.Mult())
	}
}

// TestGainedIsOnlyMultipliedPoints: the pop-ups show only what the combo
// multiplied, on the tick it scored.
func TestGainedIsOnlyMultipliedPoints(t *testing.T) {
	w := comboWorld()
	hops(w, Up, 2)
	if w.Gained != 0 {
		t.Errorf("gained %d at x1", w.Gained)
	}
	hops(w, Up, 1)
	if w.Gained != 2*config.PointsHop {
		t.Errorf("gained %d on the third hop, want %d", w.Gained, 2*config.PointsHop)
	}
	w.Update(None)
	if w.Gained != 0 {
		t.Errorf("gained %d a tick later", w.Gained)
	}
}

func TestNoComboWithoutTheLevelsSay(t *testing.T) {
	w := newWorld(1)
	if ev := hops(w, Up, 9); slices.Contains(ev, ComboUp) || w.Combo.Hops != 0 {
		t.Errorf("combo %+v in a level without it", w.Combo)
	}
	if want := 9 * config.PointsHop; w.Session.Score != want {
		t.Errorf("score %d, want %d", w.Session.Score, want)
	}
}

// TestComboEndsWhenStill: ComboTicks without a forward hop end it.
func TestComboEndsWhenStill(t *testing.T) {
	w := comboWorld()
	hops(w, Up, 1)
	run(w, None, config.ComboTicks-2)
	if w.Combo.Hops != 1 {
		t.Fatalf("combo %+v ended early", w.Combo)
	}
	run(w, None, 1)
	if w.Combo.Hops != 0 {
		t.Errorf("combo %+v still going after %d ticks", w.Combo, config.ComboTicks)
	}
}

// TestSidewaysHopsNeitherAddNorSave: sideways hops don't count, and don't
// keep the combo going.
func TestSidewaysHopsNeitherAddNorSave(t *testing.T) {
	w := comboWorld()
	hops(w, Up, 1)
	hops(w, Left, 4)
	if w.Combo.Hops != 1 {
		t.Fatalf("combo %+v after sideways hops, want 1", w.Combo)
	}
	hops(w, Right, config.ComboTicks/config.HopTicks)
	if w.Combo.Hops != 0 {
		t.Errorf("combo %+v kept going by sideways hops", w.Combo)
	}
}

func TestHopBackEndsTheCombo(t *testing.T) {
	w := comboWorld()
	hops(w, Up, 4)
	hops(w, Down, 1)
	if w.Combo.Hops != 0 {
		t.Errorf("combo %+v after a hop back", w.Combo)
	}
}

func TestDeathEndsTheCombo(t *testing.T) {
	w := comboWorld()
	hops(w, Up, 4)
	w.die(Squashed)
	if w.Combo.Hops != 0 {
		t.Errorf("combo %+v after a death", w.Combo)
	}
}

// TestRidingPausesTheCombo: waiting on a platform doesn't end the combo
// (D38).
func TestRidingPausesTheCombo(t *testing.T) {
	x0 := (config.StartCol - 1) * cellUnits
	w := riverWorld(config.LogShort, false, x0, 1, config.SubPx)
	w.Level.Combo = true
	w.Player = newPlayer(config.StartCol, riverRow+1)
	w.furthest = riverRow + 1
	hops(w, Up, 1)
	run(w, None, 2*config.ComboTicks)
	if w.Dying != 0 || w.Combo.Hops != 1 {
		t.Fatalf("combo %+v, dying %d after riding", w.Combo, w.Dying)
	}
	hops(w, Up, 1)
	if w.Combo.Hops != 2 {
		t.Errorf("combo %+v after riding then hopping on, want 2", w.Combo)
	}
}

// TestComboMultipliesTheNook: the nook's points, not the time bonus, are
// multiplied when it is reached with the combo going, and the combo is
// then over.
func TestComboMultipliesTheNook(t *testing.T) {
	w := bankWorld(config.Nooks[1])
	w.Level.Combo = true
	w.Combo = Combo{Hops: 5, Left: config.ComboTicks} // x2, and x3 with the hop to the bank
	w.TicksLeft = 20*config.TPS + 20                  // 20 whole seconds left when it lands
	hops(w, Up, 1)
	want := 3*config.PointsHop + 3*config.PointsGoal + 20*config.PointsPerSec
	if w.Session.Score != want {
		t.Errorf("score %d, want %d", w.Session.Score, want)
	}
	if w.Gained != 3*config.PointsHop+3*config.PointsGoal {
		t.Errorf("gained %d, want %d", w.Gained, 3*config.PointsHop+3*config.PointsGoal)
	}
	if w.Combo.Hops != 0 {
		t.Errorf("combo %+v after the nook", w.Combo)
	}
}
