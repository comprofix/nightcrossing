package config

import (
	"strings"
	"testing"
)

// TestBonusLayouts checks each mode's bonus level: parked cars only on
// road rows with no traffic, each row as wide as the field, each car's
// letter repeated for exactly the cells it takes, and the start cell clear
// (D61).
func TestBonusLayouts(t *testing.T) {
	for _, m := range Modes {
		b := m.Bonus
		if b == nil {
			continue
		}
		if b.Fireflies < b.AtOnce || (b.AtOnce > 0) != (b.Fireflies > 0) || b.Size() == 0 || b.Secs <= 0 ||
			b.OnRoad < 0 || b.OnRoad > b.Fireflies {
			t.Errorf("%s: %d fireflies, %d on the road, %d at once, %d fruit, %d s", m.ID, b.Fireflies, b.OnRoad,
				b.AtOnce, len(b.Fruit), b.Secs)
		}
		for r, s := range b.Parked {
			if s == "" {
				continue
			}
			lane := b.Layout.Lanes[r]
			if lane.Kind != Road || len(lane.Pattern) > 0 {
				t.Errorf("%s: parked cars on row %d, which isn't a road without traffic", m.ID, r)
			}
			if len(s) != Cols {
				t.Errorf("%s: row %d is %d cells, want %d", m.ID, r, len(s), Cols)
			}
			for c := 0; c < len(s); {
				if s[c] == '.' {
					c++
					continue
				}
				e, ok := ParkedCars[s[c]]
				if !ok {
					t.Fatalf("%s: row %d, column %d: %q is no car", m.ID, r, c, s[c])
				}
				n := ParkedCells(e)
				if c+n > len(s) || s[c:c+n] != strings.Repeat(s[c:c+1], n) {
					t.Errorf("%s: row %d, column %d: a %s takes %d cells", m.ID, r, c, e.ID, n)
				}
				c += n
			}
		}
		if s := b.Parked[StartRow-1]; s != "" && s[StartCol] != '.' {
			t.Errorf("%s: a parked car right above the start", m.ID)
		}
		lv, ok := BonusLevel(m)
		if !ok || lv.Bonus != b || lv.TimerTicks != b.Secs*TPS || lv.FireflyPct != 0 || lv.FruitPts != 0 {
			t.Errorf("%s: bonus level %+v", m.ID, lv)
		}
		if lv.FireflyOn[HostPads] || !lv.FireflyOn[HostLogs] {
			t.Errorf("%s: fireflies on %v, want logs and never lily pads", m.ID, lv.FireflyOn)
		}
	}
	if Stroll.Bonus != FireflyNight {
		t.Error("Stroll's bonus level isn't firefly night")
	}
}

// TestBonusYards checks each bonus level's goods yard and fruit: rakes
// only on track rows, never running into each other as they shunt, at
// most one last train; fruit on a row it can lie on, inside the field,
// each one a real fruit (DESIGN §6.7).
func TestBonusYards(t *testing.T) {
	for _, m := range Modes {
		b := m.Bonus
		if b == nil {
			continue
		}
		last := 0
		for i, r := range b.Yard {
			if b.Layout.Lanes[r.Row].Kind != Rail || r.Cars <= 0 {
				t.Errorf("%s: rake %d on row %d, with %d carriages", m.ID, i, r.Row, r.Cars)
			}
			if r.Last {
				last++
			}
			if r.Engine && r.Facing != 1 && r.Facing != -1 || r.Shunt != 0 && !r.Engine {
				t.Errorf("%s: rake %d: engine %v facing %d, shunting %d: only a rake with an engine shunts",
					m.ID, i, r.Engine, r.Facing, r.Shunt)
			}
			for j, o := range b.Yard[:i] {
				lo, hi := min(r.Col, r.Col+r.Shunt), max(r.Col, r.Col+r.Shunt)+r.Cells()
				olo, ohi := min(o.Col, o.Col+o.Shunt), max(o.Col, o.Col+o.Shunt)+o.Cells()
				if o.Row == r.Row && lo < ohi && olo < hi {
					t.Errorf("%s: rakes %d and %d on row %d can run into each other", m.ID, j, i, r.Row)
				}
			}
		}
		if last > 1 {
			t.Errorf("%s: %d last trains", m.ID, last)
		}
		for i, f := range b.Fruit {
			if f.Fruit < 0 || f.Fruit >= len(Fruits) || f.Side == FruitYard && len(b.Yard) == 0 || f.Side > FruitRoad {
				t.Errorf("%s: fruit %d is %d, on side %d, with %d rakes", m.ID, i, f.Fruit, f.Side, len(b.Yard))
			}
		}
	}
	if LastTrain.Bonus != LastTrainHome || LastTrainHome.Size() != 12 {
		t.Error("Last Train's bonus level isn't the last train home, with 12 fruit")
	}
	river := 0
	for _, f := range LastTrainHome.Fruit {
		if f.Side == FruitRiver {
			river++
		}
	}
	if river*2 != len(LastTrainHome.Fruit) {
		t.Errorf("the last train home has %d of its %d fruit on the river, want half", river, len(LastTrainHome.Fruit))
	}
}

// TestEmptyRoads: Rush Hour's bonus level is empty roads at dawn, its
// road empty but for the sweeper, and its fruit all on the road and verge
// (DESIGN §6.7).
func TestEmptyRoads(t *testing.T) {
	b := EmptyRoads
	if RushHour.Bonus != b || !b.Sweeper || !b.Dawn || b.Size() != 12 || len(b.Yard) != 0 {
		t.Fatalf("Rush Hour's bonus level %+v", b)
	}
	for r, l := range b.Layout.Lanes {
		if l.Kind == Road && len(l.Pattern) > 0 {
			t.Errorf("traffic on row %d", r)
		}
	}
	for i, f := range b.Fruit {
		if f.Side != FruitRoad {
			t.Errorf("fruit %d isn't on the road", i)
		}
	}
}
