package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

const railRow = 7

// trainWorld is an all-safe level with the track on row 7 and trains on.
func trainWorld(seed uint64) *World {
	lv := testLevel()
	lv.Lanes[railRow].Kind = config.Rail
	lv.Trains, lv.Train = true, config.TrainsOn(1)
	return New(session.New(config.LastTrain, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
}

// until runs w with nothing pressed until e happens, and returns how many
// ticks that took, failing after limit.
func until(t *testing.T, w *World, e Event, limit int) int {
	t.Helper()
	for n := 1; n <= limit; n++ {
		if slices.Contains(w.Update(None), e) {
			return n
		}
	}
	t.Fatalf("no %v in %d ticks", e, limit)
	return 0
}

func TestNoTrainsInStroll(t *testing.T) {
	w := New(session.New(config.Stroll, config.StartLives), config.StrollLevels.Level(1), rand.New(rand.NewPCG(1, 1)))
	if w.Rail.On {
		t.Fatal("trains run in Stroll")
	}
	for range 30 * config.TPS {
		for _, e := range w.Update(None) {
			if e == TrainWarning || e == TrainComing {
				t.Fatalf("event %v in Stroll", e)
			}
		}
	}
}

func TestLastTrainHasTrainsFromLevel1(t *testing.T) {
	for _, n := range []int{1, 8, 20} {
		if !config.LastTrain.Levels.Level(n).Trains {
			t.Errorf("level %d has no trains", n)
		}
	}
}

// TestTrainCycle: a quiet spell of seeded length, a warning of
// TrainWarnTicks, a train that crosses the whole field, then quiet again.
func TestTrainCycle(t *testing.T) {
	dirs := map[int]bool{}
	for seed := uint64(1); seed <= 12; seed++ {
		w := trainWorld(seed)
		if !w.Rail.On || w.Rail.Row != railRow || w.Rail.Phase != RailQuiet {
			t.Fatalf("rail %+v at the start", w.Rail)
		}
		quiet := until(t, w, TrainWarning, config.TrainFirst.QuietMax)
		if quiet < config.TrainFirst.QuietMin {
			t.Errorf("seed %d: quiet for %d ticks, under %d", seed, quiet, config.TrainFirst.QuietMin)
		}
		if warn := until(t, w, TrainComing, config.TrainWarnTicks); warn != config.TrainWarnTicks {
			t.Errorf("seed %d: warning of %d ticks, want %d", seed, warn, config.TrainWarnTicks)
		}
		dirs[w.Rail.Dir] = true
		pass := until(t, w, TrainGone, 10*config.TPS)
		want := (fieldWidth + trainUnits + config.TrainFirst.Speed - 1) / config.TrainFirst.Speed
		if pass < want-1 || pass > want+1 {
			t.Errorf("seed %d: passing took %d ticks, want about %d", seed, pass, want)
		}
		if w.Rail.Phase != RailQuiet {
			t.Errorf("seed %d: phase %v after the train", seed, w.Rail.Phase)
		}
	}
	if !dirs[1] || !dirs[-1] {
		t.Errorf("trains only ran %v", dirs)
	}
}

func TestTrainStrikesTheHeroOnTheTrack(t *testing.T) {
	w := trainWorld(3)
	w.Player = newPlayer(config.Cols/2, railRow)
	until(t, w, TrainComing, config.TrainFirst.QuietMax+config.TrainWarnTicks)
	if w.Dying != 0 {
		t.Fatal("the hero died before the train came")
	}
	until(t, w, Struck, 3*config.TPS)
	if w.Player.Death != Struck {
		t.Errorf("death %v, want struck", w.Player.Death)
	}
}

func TestTrainMissesTheHeroBesideTheTrack(t *testing.T) {
	w := trainWorld(3)
	w.Player = newPlayer(config.Cols/2, railRow+1)
	until(t, w, TrainComing, config.TrainFirst.QuietMax+config.TrainWarnTicks)
	until(t, w, TrainGone, 5*config.TPS)
	if w.Dying != 0 {
		t.Errorf("the hero beside the track died: %v", w.Player.Death)
	}
}

func TestTrainsSameSeedSameTimes(t *testing.T) {
	times := func() []int64 {
		w := trainWorld(9)
		var out []int64
		for range 60 * config.TPS {
			for _, e := range w.Update(None) {
				if e == TrainWarning || e == TrainComing || e == TrainGone {
					out = append(out, w.Tick)
				}
			}
		}
		return out
	}
	a, b := times(), times()
	if !slices.Equal(a, b) || len(a) < 9 {
		t.Errorf("two runs: %v and %v", a, b)
	}
}

// TestTopLevelTrainsAreFaster: the train on the top level crosses the
// field in about half the time it takes on level 1 (D17).
func TestTopLevelTrainsAreFaster(t *testing.T) {
	pass := func(n int) int {
		w := trainWorld(5)
		w.Rail.Timing = config.TrainsOn(n)
		until(t, w, TrainComing, config.TrainFirst.QuietMax+config.TrainWarnTicks)
		return until(t, w, TrainGone, 10*config.TPS)
	}
	first, top := pass(1), pass(config.TrainTopLevel)
	if top*2 > first+2 {
		t.Errorf("level 1 train passes in %d ticks, top level's in %d: want about half", first, top)
	}
}
