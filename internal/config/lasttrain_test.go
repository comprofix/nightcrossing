package config

import "testing"

// TestLastTrainBringsItsMechanicsIn: trains from level 1, the combo from
// level 2, day and night from level 3, each with a hint where it starts,
// and none of them in Stroll (D27).
func TestLastTrainBringsItsMechanicsIn(t *testing.T) {
	for _, c := range []struct {
		n                    int
		trains, combo, cycle bool
	}{
		{1, true, false, false},
		{2, true, true, false},
		{3, true, true, true},
		{8, true, true, true},
		{30, true, true, true},
	} {
		lv := LastTrain.Levels.Level(c.n)
		if lv.Trains != c.trains || lv.Combo != c.combo || lv.DayCycle != c.cycle {
			t.Errorf("level %d: trains %v combo %v cycle %v, want %v %v %v",
				c.n, lv.Trains, lv.Combo, lv.DayCycle, c.trains, c.combo, c.cycle)
		}
	}
	for n := 1; n <= 3; n++ {
		if LastTrain.Levels.Level(n).Hint == "" {
			t.Errorf("level %d has no hint for what it brings in", n)
		}
	}
	for _, n := range []int{1, 3, 8, 30} {
		if lv := StrollLevels.Level(n); lv.Trains || lv.Combo || lv.DayCycle {
			t.Errorf("Stroll level %d has Last Train's mechanics", n)
		}
	}
}

func TestLastTrainIsInTheMenu(t *testing.T) {
	if len(Modes) != 2 || Modes[0].ID != Stroll.ID || Modes[1].ID != LastTrain.ID {
		t.Errorf("modes %v, want Stroll then Last Train", Modes)
	}
}

// TestDayCycle: each level starts at dawn and the clock only moves on as
// the nooks fill, ending at night (D39).
func TestDayCycle(t *testing.T) {
	if DayCycle[0] != Dawn || DayCycle[len(DayCycle)-1] != Night {
		t.Errorf("cycle %v: want dawn to night", DayCycle)
	}
	for i := 1; i < len(DayCycle); i++ {
		if DayCycle[i] < DayCycle[i-1] {
			t.Errorf("the clock turns back at %d nooks: %v", i, DayCycle)
		}
	}
}
