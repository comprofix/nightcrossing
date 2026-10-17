package config

import "testing"

// TestStormBringsItsMechanicsIn: everything from Rush Hour from level 1,
// currents from level 1 and gusts from level 2, with a hint for each
// (D45, D50).
func TestStormBringsItsMechanicsIn(t *testing.T) {
	for _, n := range []int{1, 2, 8, 30} {
		lv := Storm.Levels.Level(n)
		if !lv.Trains || !lv.Combo || !lv.DayCycle {
			t.Errorf("level %d: trains %v combo %v cycle %v, want all from level 1", n, lv.Trains, lv.Combo, lv.DayCycle)
		}
		for _, k := range []EventKind{Wave, Fog, Rain} {
			if !lv.Events[k].On() {
				t.Errorf("level %d: event %d is off, want it from level 1", n, k)
			}
		}
		if got := lv.Events[Gust].On(); got != (n >= 2) {
			t.Errorf("level %d: gusts %v, want them from level 2", n, got)
		}
		if got := lv.Events[Flood].On(); got != (n >= 3) {
			t.Errorf("level %d: floods %v, want them from level 3", n, got)
		}
		if !lv.Stormy {
			t.Errorf("level %d: no storm sky", n)
		}
	}
	for n := 1; n <= 3; n++ {
		if Storm.Levels.Level(n).Hint == "" {
			t.Errorf("level %d has no hint for what it brings in", n)
		}
	}
	for _, m := range []Mode{Stroll, LastTrain, RushHour} {
		for _, n := range []int{1, 8, 30} {
			lv := m.Levels.Level(n)
			if lv.Events[Gust].On() || lv.Events[Flood].On() || lv.Stormy || currentRows(lv) > 0 {
				t.Errorf("%s level %d has Storm's mechanics", m.Name, n)
			}
		}
	}
}

// currentRows is how many of lv's rows have a current.
func currentRows(lv Level) int {
	n := 0
	for _, l := range lv.Lanes {
		if l.Current != 0 {
			n++
		}
	}
	return n
}

// TestStormCurrents: one current row on levels 1–3, two on 4–6, three on
// 7–8, only on river rows of logs and ships, getting stronger, and scaled
// with the level's speed; the loops keep level 8's (D46, D50).
func TestStormCurrents(t *testing.T) {
	m := Storm.Levels
	want := []int{1, 1, 1, 2, 2, 2, 3, 3}
	strongest := 0
	for n := 1; n <= len(m.Levels); n++ {
		lv := m.Level(n)
		if got := currentRows(lv); got != want[n-1] {
			t.Errorf("level %d: %d current rows, want %d", n, got, want[n-1])
		}
		top := 0
		for r, l := range lv.Lanes {
			if l.Current == 0 {
				continue
			}
			if l.Kind != River {
				t.Errorf("level %d row %d: a current on a lane that isn't river", n, r)
			}
			for _, s := range l.Pattern {
				if s.Entity == Pads2 || s.Entity == Pads3 {
					t.Errorf("level %d row %d: a current on a row of lily pads", n, r)
				}
			}
			top = max(top, abs(l.Current))
		}
		if top < strongest {
			t.Errorf("level %d: currents up to %d, weaker than the level before's %d", n, top, strongest)
		}
		strongest = top
	}
	if c := m.Level(1).Lanes[3].Current; c >= 0 {
		t.Errorf("level 1's current is %d, want it against the flow", c)
	}

	def, speed := m.Def(1)
	if got, want := m.Level(1).Lanes[def.Currents[0].Row].Current, def.Currents[0].Push*speed/100; got != want {
		t.Errorf("level 1's current %d, want %d", got, want)
	}
	last := m.Levels[len(m.Levels)-1]
	for n := len(m.Levels) + 1; n <= len(m.Levels)+2*LoopLevels; n++ {
		def, speed := m.Def(n)
		lv := m.Level(n)
		for _, c := range last.Currents {
			if got, want := lv.Lanes[c.Row].Current, c.Push*speed/100; got != want {
				t.Errorf("loop level %d row %d: current %d, want level 8's at %d%%: %d", n, c.Row, got, speed, want)
			}
		}
		if len(def.Currents) != len(last.Currents) {
			t.Errorf("loop level %d: %d current rows, want level 8's %d", n, len(def.Currents), len(last.Currents))
		}
	}
}

// TestGustsRamp: gust spells come more often, last longer and push on a
// quicker beat each level, to the top (D47).
func TestGustsRamp(t *testing.T) {
	first := EventsOn(Gust, 2, 2)
	top := EventsOn(Gust, 2, EventTopLevel)
	if first != Events[Gust].First || top != Events[Gust].Top {
		t.Errorf("first %+v top %+v, want %+v and %+v", first, top, Events[Gust].First, Events[Gust].Top)
	}
	for _, g := range []EventTiming{first, top} {
		if g.Beat <= GustTellTicks || g.Beat >= g.Last {
			t.Errorf("%+v: the beat must leave room for the tell and come within the spell", g)
		}
	}
	if GustMessage(-1) == GustMessage(1) {
		t.Error("a gust's warning doesn't say which way it blows")
	}
}

// TestFloodsRamp: floods come more often, last longer and take more of the
// verge each level, to the top: about 4 cells rising to 12, never more
// than half (D48).
func TestFloodsRamp(t *testing.T) {
	first := EventsOn(Flood, 3, 3)
	top := EventsOn(Flood, 3, EventTopLevel)
	if first.Size != 4 || top.Size != 12 || top.Size > FloodMostCells {
		t.Errorf("flood sizes %d to %d, want 4 to 12, at most %d", first.Size, top.Size, FloodMostCells)
	}
	if mid := EventsOn(Flood, 3, 5); mid.Size <= first.Size || mid.Size >= top.Size {
		t.Errorf("level 5 floods %d cells, want between %d and %d", mid.Size, first.Size, top.Size)
	}
	for _, f := range []EventTiming{first, top} {
		if f.Last <= 2*FloodPuddleTicks || f.Last <= 2*FloodSurgeTicks {
			t.Errorf("%+v: too short for its puddles and surge", f)
		}
	}
}
