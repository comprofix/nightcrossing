package config

import (
	"slices"
	"testing"
)

func TestPlayfieldFillsTheScreen(t *testing.T) {
	p := Project
	if w := 2*FieldMargin + Cols*Cell; w != p.ScreenWidth {
		t.Errorf("margins and grid are %d wide, screen is %d", w, p.ScreenWidth)
	}
	if h := HUDHeight + Rows*Cell + HintHeight; h != p.ScreenHeight {
		t.Errorf("status bar, grid and hint strip are %d high, screen is %d", h, p.ScreenHeight)
	}
}

func TestSkills(t *testing.T) {
	if len(Skills) != 4 {
		t.Fatalf("%d skills, want 4", len(Skills))
	}
	ids := map[string]bool{}
	for i, s := range Skills {
		if s.ID == "" || s.Name == "" || s.Desc == "" {
			t.Errorf("skill %d is missing an ID, name or description: %+v", i, s)
		}
		if ids[s.ID] {
			t.Errorf("skill ID %q is used twice", s.ID)
		}
		ids[s.ID] = true
		if s.SpeedPct <= 0 || s.TimerSecs <= 0 || s.HitMargin < 0 || 2*s.HitMargin >= Cell {
			t.Errorf("skill %s has unusable tuning: %+v", s.ID, s)
		}
		if i > 0 {
			prev := Skills[i-1]
			if s.SpeedPct < prev.SpeedPct || s.TimerSecs > prev.TimerSecs || s.HitMargin > prev.HitMargin {
				t.Errorf("skill %s is easier than %s, which comes before it", s.ID, prev.ID)
			}
		}
	}
}

func TestEntities(t *testing.T) {
	longest := 0
	ids := map[string]bool{}
	for _, e := range Entities() {
		if e.ID == "" || e.Length <= 0 {
			t.Errorf("entity %+v has no ID or length", e)
		}
		if ids[e.ID] {
			t.Errorf("entity ID %q is used twice", e.ID)
		}
		ids[e.ID] = true
		longest = max(longest, e.Length)
	}
	if longest != Longest {
		t.Errorf("Longest is %d, the longest entity is %d", Longest, longest)
	}
	// The hero must fit on the smallest platform with room to spare.
	for _, p := range Platforms() {
		if p.Length < 2*Cell {
			t.Errorf("platform %s is %dpx long, too short to land on", p.ID, p.Length)
		}
	}
}

func TestDiveCycle(t *testing.T) {
	if DiveWarnTicks < 2*DiveFlashTicks {
		t.Errorf("the warning (%d ticks) is too short for a whole flash (%d on, %d off)", DiveWarnTicks, DiveFlashTicks, DiveFlashTicks)
	}
	// A hero who sees the warning start must have time to hop off.
	if DiveWarnTicks+DiveSinkTicks < 2*HopTicks {
		t.Errorf("only %d ticks from the warning to going under", DiveWarnTicks+DiveSinkTicks)
	}
	if SinkWarnTicks+SinkBreakTicks+SinkPieceTicks < 2*HopTicks {
		t.Errorf("only %d ticks from a ship's warning to its bow going under", SinkWarnTicks+SinkBreakTicks+SinkPieceTicks)
	}
	if SinkAfterMinTicks <= 0 || SinkAfterMaxTicks < SinkAfterMinTicks {
		t.Errorf("ships float %d–%d ticks before sinking", SinkAfterMinTicks, SinkAfterMaxTicks)
	}
	for _, s := range Ships() {
		if s.Length%Cell != 0 {
			t.Errorf("ship %s is %dpx long, not whole cells, so it can't go under piece by piece", s.ID, s.Length)
		}
	}
}

func TestLevels(t *testing.T) {
	for i, lv := range Levels {
		for r, l := range lv.Lanes {
			if (r == GoalRow || r == StartRow) && l.Kind != Safe {
				t.Errorf("level %d row %d must be safe", i+1, r)
			}
			if l.Kind == Safe || l.Kind == Rail {
				if len(l.Pattern) != 0 {
					t.Errorf("level %d row %d: a safe or rail row has a pattern", i+1, r)
				}
				continue
			}
			if l.Dir != 1 && l.Dir != -1 {
				t.Errorf("level %d row %d: direction %d", i+1, r, l.Dir)
			}
			if l.Speed <= 0 || len(l.Pattern) == 0 {
				t.Errorf("level %d row %d: no speed or pattern", i+1, r)
			}
			steady := false
			for _, s := range l.Pattern {
				if l.Kind == Road && s.Gap < Cell {
					t.Errorf("level %d row %d: a gap of %dpx is too small to stand in", i+1, r, s.Gap)
				}
				if s.Dives && !slices.Contains(Pads(), s.Entity) {
					t.Errorf("level %d row %d: %s dives, but only lily pads can", i+1, r, s.Entity.ID)
				}
				if s.Sinks && !slices.Contains(Ships(), s.Entity) {
					t.Errorf("level %d row %d: %s sinks, but only ships can", i+1, r, s.Entity.ID)
				}
				if s.Gap <= 0 {
					t.Errorf("level %d row %d: a gap of %dpx", i+1, r, s.Gap)
				}
				steady = steady || (!s.Dives && !s.Sinks)
			}
			// A river row always has a platform that stays up.
			if !steady {
				t.Errorf("level %d row %d: every platform dives or sinks", i+1, r)
			}
			for _, s := range l.Pattern {
				if slices.Contains(Vehicles(), s.Entity) != (l.Kind == Road) {
					t.Errorf("level %d row %d: %s does not belong in this kind of lane", i+1, r, s.Entity.ID)
				}
			}
		}
	}
}
