package config

import "testing"

// TestSchoolRunLevels: one toadlet on levels 1–4 and two from level 5,
// through the loops; a second a toadlet on the timer; Storm's mechanics,
// storms and all, from level 1 with no taught first storm; and the hints
// on levels 1, 2 and 5 (D53, D54).
func TestSchoolRunLevels(t *testing.T) {
	m := SchoolRun.Levels
	hints := map[int]bool{1: true, 2: true, 5: true}
	for n := 1; n <= len(m.Levels)+2*LoopLevels; n++ {
		lv := m.Level(n)
		want := 1
		if n >= 5 {
			want = 2
		}
		if lv.Toadlets != want {
			t.Errorf("level %d: %d toadlets, want %d", n, lv.Toadlets, want)
		}
		if lv.TimerTicks != TimerTicks(n)+want*ToadletSecs*TPS {
			t.Errorf("level %d: timer %d ticks, want %d", n, lv.TimerTicks, TimerTicks(n)+want*ToadletSecs*TPS)
		}
		s := lv.Events[StormFront]
		if !s.On() || s.Beat == 0 || s.Size == 0 || !lv.Events[Rain].On() || !lv.Combo || !lv.DayCycle || !lv.Trains {
			t.Errorf("level %d: not all of Storm from level 1", n)
		}
		if lv.FirstStorm != AnyStorm {
			t.Errorf("level %d: first storm %v, want the mix", n, lv.FirstStorm)
		}
		if (lv.Hint != "") != hints[n] {
			t.Errorf("level %d: hint %q", n, lv.Hint)
		}
	}
	if lv := Storm.Levels.Level(2); lv.FirstStorm != DryStorm {
		t.Errorf("Storm's level 2 starts with %v, want a dry storm", lv.FirstStorm)
	}
}
