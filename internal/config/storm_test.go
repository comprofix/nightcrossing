package config

import "testing"

// TestStormBringsItsMechanicsIn: everything from Rush Hour from level 1,
// but plain rain from level 2; currents and storms from level 1, dry
// storms from level 2 and floods from level 3, with a hint for each (D45,
// D50, D51).
func TestStormBringsItsMechanicsIn(t *testing.T) {
	for _, n := range []int{1, 2, 3, 8, 30} {
		lv := Storm.Levels.Level(n)
		if !lv.Trains || !lv.Combo || !lv.DayCycle {
			t.Errorf("level %d: trains %v combo %v cycle %v, want all from level 1", n, lv.Trains, lv.Combo, lv.DayCycle)
		}
		for _, k := range []EventKind{Wave, Fog, StormFront} {
			if !lv.Events[k].On() {
				t.Errorf("level %d: event %d is off, want it from level 1", n, k)
			}
		}
		if got := lv.Events[Rain].On(); got != (n >= 2) {
			t.Errorf("level %d: plain rain %v, want it from level 2", n, got)
		}
		s := lv.Events[StormFront]
		if got := s.Beat > 0; got != (n >= 2) {
			t.Errorf("level %d: wind beat %d, want dry storms from level 2", n, s.Beat)
		}
		if got := s.Size > 0; got != (n >= 3) {
			t.Errorf("level %d: flood size %d, want floods from level 3", n, s.Size)
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
			if lv.Events[StormFront].On() || currentRows(lv) > 0 {
				t.Errorf("%s level %d has Storm's mechanics", m.Name, n)
			}
		}
	}
}

// TestFirstStorms: level 1's storms are all wet, so it has no first kind
// to set; level 2 starts with a dry storm and level 3 with a wet one, to
// teach each; the rest are the seeded mix (D51).
func TestFirstStorms(t *testing.T) {
	for n, want := range map[int]StormKind{1: AnyStorm, 2: DryStorm, 3: WetStorm, 4: AnyStorm, 8: AnyStorm, 12: AnyStorm} {
		if got := Storm.Levels.Level(n).FirstStorm; got != want {
			t.Errorf("level %d: first storm %d, want %d", n, got, want)
		}
	}
}

// TestStormShare: a storm takes about a quarter of level 1 and about half
// at the top, never more, and the clear spells shorten as the storms
// lengthen (D51).
func TestStormShare(t *testing.T) {
	share := func(n int) float64 {
		s := Storm.Levels.Level(n).Events[StormFront]
		storm := float64(s.Last+s.LastMax)/2 + float64(Events[StormFront].WarnTicks)
		clear := float64(s.QuietMin+s.QuietMax) / 2
		return storm / (storm + clear)
	}
	if got := share(1); got < 0.2 || got > 0.3 {
		t.Errorf("level 1: storms %.0f%% of the time, want about a quarter", got*100)
	}
	if got := share(EventTopLevel); got < 0.4 || got > 0.55 {
		t.Errorf("top level: storms %.0f%% of the time, want about half, at most", got*100)
	}
	for n := 2; n <= EventTopLevel+3; n++ {
		if share(n) < share(n-1) {
			t.Errorf("level %d: storms %.2f of the time, less than level %d's %.2f", n, share(n), n-1, share(n-1))
		}
	}
}

// TestPlainRainIsRarer: in Storm the plain rain comes from level 2, its
// quiet half as long again as Rush Hour's rain would be, so the clear
// spells stay mostly clear (D45).
func TestPlainRainIsRarer(t *testing.T) {
	for _, n := range []int{2, 3, 8} {
		storm, rush := Storm.Levels.Level(n).Events[Rain], EventsOn(Rain, 2, n)
		if storm.QuietMin != rush.QuietMin*3/2 || storm.QuietMax != rush.QuietMax*3/2 || storm.Last != rush.Last {
			t.Errorf("level %d: Storm's rain %+v, the usual %+v", n, storm, rush)
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

// TestWindRamps: the wind pushes every 4 s on the level dry storms
// arrive, every 3 s on the next, and on a quicker beat each level to 2 s
// at the top, always leaving room for its tell (D47).
func TestWindRamps(t *testing.T) {
	want := map[int]int{1: 0, 2: 4 * TPS, 3: 3 * TPS, EventTopLevel: 2 * TPS, EventTopLevel + 5: 2 * TPS}
	for n, b := range want {
		if got := Storm.Levels.Level(n).Events[StormFront].Beat; got != b {
			t.Errorf("level %d: a push every %d ticks, want %d", n, got, b)
		}
	}
	for n := 4; n <= EventTopLevel; n++ {
		b, before := WindBeat(2, n), WindBeat(2, n-1)
		if b > before || b <= WindTellTicks {
			t.Errorf("level %d: a beat of %d after %d", n, b, before)
		}
	}
	if StormMessage(false, -1) == StormMessage(false, 1) || StormMessage(true, 1) == StormMessage(false, 1) {
		t.Error("a storm's warning doesn't say its kind and which way it blows")
	}
}

// TestFloodsRamp: a wet storm's flood takes about 4 verge cells on level 3
// rising to 12 at the top, never more than half, and it comes and drains
// well inside the shortest storm (D48).
func TestFloodsRamp(t *testing.T) {
	first, top := Storm.Levels.Level(3).Events[StormFront], Storm.Levels.Level(EventTopLevel).Events[StormFront]
	if first.Size != 4 || top.Size != 12 || top.Size > FloodMostCells {
		t.Errorf("flood sizes %d to %d, want 4 to 12, at most %d", first.Size, top.Size, FloodMostCells)
	}
	if mid := Storm.Levels.Level(5).Events[StormFront]; mid.Size <= first.Size || mid.Size >= top.Size {
		t.Errorf("level 5 floods %d cells, want between %d and %d", mid.Size, first.Size, top.Size)
	}
	if room := first.Last - StormFloodDelay; room <= 2*FloodPuddleTicks || room <= 2*FloodSurgeTicks {
		t.Errorf("a %d-tick storm leaves %d ticks for its flood: too short for its puddles and surge", first.Last, room)
	}
}
