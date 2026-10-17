package config

import "testing"

// TestStrollPickups: Stroll brings fruit in on level 2 and fireflies one
// kind of platform at a time from level 4, each with a hint (D60).
func TestStrollPickups(t *testing.T) {
	type want struct {
		fruit, kind, pct int
		on               [NumHosts]bool
		hint             string
	}
	for n, w := range map[int]want{
		1:  {0, 0, 0, [NumHosts]bool{}, "RIDE THE LOGS TO THE NOOKS ON THE FAR BANK"},
		2:  {200, 1, 0, [NumHosts]bool{}, "FRUIT IN A NOOK IS A BONUS"},
		3:  {300, 2, 0, [NumHosts]bool{}, "LILY PADS, A BARGE AND THE FERRY JOIN THE LOGS"},
		4:  {400, 3, 30, [NumHosts]bool{true, false, false}, "CATCH A FIREFLY FOR AN EXTRA LIFE"},
		5:  {500, 4, 35, [NumHosts]bool{true, true, false}, "LILY PADS THAT FLASH ARE ABOUT TO DIVE"},
		7:  {700, 6, 45, [NumHosts]bool{true, true, true}, "A SINKING SHIP: RUN TO ITS STERN OR HOP OFF"},
		8:  {800, 7, 50, [NumHosts]bool{true, true, true}, ""},
		10: {800, 7, 50, [NumHosts]bool{true, true, true}, ""},
		20: {800, 7, 50, [NumHosts]bool{true, true, true}, ""}, // the loops stay at the top
	} {
		lv := StrollLevels.Level(n)
		if lv.FruitPts != w.fruit || (w.fruit > 0 && lv.Fruit != w.kind) {
			t.Errorf("level %d: fruit %d (%s), want %d (%s)", n, lv.FruitPts, Fruits[lv.Fruit], w.fruit, Fruits[w.kind])
		}
		if lv.FireflyPct != w.pct || lv.FireflyOn != w.on {
			t.Errorf("level %d: fireflies %d%% on %v, want %d%% on %v", n, lv.FireflyPct, lv.FireflyOn, w.pct, w.on)
		}
		if lv.Hint != w.hint {
			t.Errorf("level %d: hint %q, want %q", n, lv.Hint, w.hint)
		}
	}
}

// TestLaterModesHavePickupsFromLevel1: every mode after Stroll has fruit
// and fireflies on every platform from its level 1, and none of Stroll's
// hints for them (D60).
func TestLaterModesHavePickupsFromLevel1(t *testing.T) {
	all := [NumHosts]bool{true, true, true}
	for _, m := range Modes[1:] {
		for n := 1; n <= len(m.Levels.Levels)+LoopLevels; n++ {
			lv := m.Levels.Level(n)
			top := min(n, len(m.Levels.Levels))
			if want := PointsFruitFirst + PointsFruitStep*(top-1); lv.FruitPts != want || lv.Fruit != top-1 {
				t.Errorf("%s level %d: fruit %d (%d), want %d (%d)", m.ID, n, lv.FruitPts, lv.Fruit, want, top-1)
			}
			if lv.FireflyOn != all || lv.FireflyPct < FireflyPctFirst || lv.FireflyPct > FireflyPctTop {
				t.Errorf("%s level %d: fireflies %d%% on %v", m.ID, n, lv.FireflyPct, lv.FireflyOn)
			}
			if lv.Hint == "FRUIT IN A NOOK IS A BONUS" || lv.Hint == "CATCH A FIREFLY FOR AN EXTRA LIFE" {
				t.Errorf("%s level %d: Stroll's hint %q", m.ID, n, lv.Hint)
			}
		}
		if lv := m.Levels.Level(1); lv.FireflyPct != FireflyPctFirst {
			t.Errorf("%s level 1: fireflies %d%%, want %d%%", m.ID, lv.FireflyPct, FireflyPctFirst)
		}
		if lv := m.Levels.Level(FireflyTopLevel); lv.FireflyPct != FireflyPctTop {
			t.Errorf("%s level %d: fireflies %d%%, want %d%%", m.ID, FireflyTopLevel, lv.FireflyPct, FireflyPctTop)
		}
	}
}

// TestEveryHostIsAPlatform: each river platform rides as some host, and
// the kinds split as D60 says.
func TestEveryHostIsAPlatform(t *testing.T) {
	for e, want := range map[Entity]Host{LogShort: HostLogs, LogLong: HostLogs, Pads2: HostPads, Pads3: HostPads, Barge: HostShips, Ferry: HostShips} {
		if got := HostOf(e); got != want {
			t.Errorf("%s rides as host %d, want %d", e.ID, got, want)
		}
	}
}
