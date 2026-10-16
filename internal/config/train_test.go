package config

import "testing"

// TestTrainsRamp: the trains start slow and far apart on level 1, get
// faster and more frequent each level, and stay at their top from
// TrainTopLevel on (D17).
func TestTrainsRamp(t *testing.T) {
	if got := TrainsOn(1); got != TrainFirst {
		t.Errorf("level 1: %+v, want %+v", got, TrainFirst)
	}
	for _, n := range []int{TrainTopLevel, TrainTopLevel + 1, 40} {
		if got := TrainsOn(n); got != TrainTop {
			t.Errorf("level %d: %+v, want %+v", n, got, TrainTop)
		}
	}
	for n := 2; n <= TrainTopLevel; n++ {
		prev, cur := TrainsOn(n-1), TrainsOn(n)
		if cur.Speed <= prev.Speed || cur.QuietMin >= prev.QuietMin || cur.QuietMax >= prev.QuietMax {
			t.Errorf("level %d: %+v, not harder than level %d's %+v", n, cur, n-1, prev)
		}
		if cur.QuietMin > cur.QuietMax {
			t.Errorf("level %d: quiet %d–%d ticks", n, cur.QuietMin, cur.QuietMax)
		}
	}
}

func TestLastTrainLevelsCarryTheRamp(t *testing.T) {
	for _, n := range []int{1, 4, 8, 30} {
		if got, want := LastTrain.Levels.Level(n).Train, TrainsOn(n); got != want {
			t.Errorf("level %d: %+v, want %+v", n, got, want)
		}
	}
	if got := StrollLevels.Level(8).Train; got != (TrainTiming{}) {
		t.Errorf("Stroll has train timing %+v", got)
	}
}
