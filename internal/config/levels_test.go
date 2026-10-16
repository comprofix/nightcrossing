package config

import "testing"

func TestTimerPerLevel(t *testing.T) {
	for n, want := range map[int]int{1: 45, 2: 43, 8: 31, 13: 21, 14: 20, 30: 20} {
		if got := TimerTicks(n); got != want*TPS {
			t.Errorf("level %d: %d ticks, want %d s", n, got, want)
		}
	}
}

func TestEightLevels(t *testing.T) {
	if len(StrollLevels.Levels) != 8 {
		t.Errorf("%d levels, want 8 (D14)", len(StrollLevels.Levels))
	}
	for i, d := range StrollLevels.Levels {
		if d.Layout == nil || d.SpeedPct <= 0 || d.GapPct <= 0 || d.HeronPct < 0 || d.HeronPct >= 100 {
			t.Errorf("level %d: %+v", i+1, d)
		}
	}
	if StrollLevels.Levels[0].SpeedPct != 100 {
		t.Errorf("level 1 at %d%% speed, want the base 100%% (D13)", StrollLevels.Levels[0].SpeedPct)
	}
}

// TestLoops: after level 8 the last 3 loop, each round only faster, by
// LoopSpeedStepPct, up to the ceiling; the road gaps stay as on level 8
// (D19, D20).
func TestLoops(t *testing.T) {
	m := StrollLevels
	last := len(m.Levels)
	l8 := m.Levels[last-1]
	for n := last + 1; n <= last+60; n++ {
		def, speed := m.Def(n)
		k := n - last - 1
		want := m.Levels[last-LoopLevels+k%LoopLevels]
		if def.Layout != want.Layout || def.HeronPct != want.HeronPct {
			t.Errorf("level %d plays %s, want %s", n, def.Layout.Name, want.Layout.Name)
		}
		if def.GapPct != l8.GapPct {
			t.Errorf("level %d: gaps at %d%%, want level 8's %d%%", n, def.GapPct, l8.GapPct)
		}
		if def.Hint != "" {
			t.Errorf("level %d repeats a hint", n)
		}
		if wantSpeed := min(want.SpeedPct+(k/LoopLevels+1)*LoopSpeedStepPct, m.CeilingPct); speed != wantSpeed {
			t.Errorf("level %d at %d%%, want %d%%", n, speed, wantSpeed)
		}
	}
	if _, s := m.Def(10_000); s != m.CeilingPct {
		t.Errorf("far into the loops at %d%%, want the ceiling %d%%", s, m.CeilingPct)
	}
}

func TestLevelSpeedsAndGaps(t *testing.T) {
	lv := StrollLevels.Level(2)
	def := StrollLevels.Levels[1]
	for r, l := range def.Layout.Lanes {
		got := lv.Lanes[r]
		if len(l.Pattern) == 0 {
			continue
		}
		if want := max(1, l.Speed*def.SpeedPct/100); got.Speed != want {
			t.Errorf("row %d: speed %d, want %d", r, got.Speed, want)
		}
		for i, s := range l.Pattern {
			want := s.Gap
			if l.Kind == Road {
				want = max(Cell, s.Gap*def.GapPct/100)
			}
			if got.Pattern[i].Gap != want {
				t.Errorf("row %d spawn %d: gap %d, want %d", r, i, got.Pattern[i].Gap, want)
			}
		}
	}
	// The layout itself is left as it was.
	if StrollLevels.Levels[1].Layout.Lanes[9].Pattern[0].Gap != strollRoad[9].Pattern[0].Gap {
		t.Error("making a level changed its layout")
	}
}

// TestSinkersComeLater: sinking ships only in the later levels, and never
// between rows of diving pads before the last level (D33). Herons only in
// the later levels too (D34).
func TestSinkersComeLater(t *testing.T) {
	const firstSinking = 7
	for i, d := range StrollLevels.Levels {
		n := i + 1
		for r, l := range d.Layout.Lanes {
			for _, s := range l.Pattern {
				if !s.Sinks {
					continue
				}
				if n < firstSinking {
					t.Errorf("level %d row %d: a sinking ship before level %d", n, r, firstSinking)
				}
				if n < len(StrollLevels.Levels) && dives(d.Layout.Lanes[r-1]) && dives(d.Layout.Lanes[r+1]) {
					t.Errorf("level %d row %d: a sinking ship between rows of diving pads", n, r)
				}
			}
		}
		if n < 6 && d.HeronPct > 0 {
			t.Errorf("level %d has herons", n)
		}
	}
}

func dives(l Lane) bool {
	for _, s := range l.Pattern {
		if s.Dives {
			return true
		}
	}
	return false
}
