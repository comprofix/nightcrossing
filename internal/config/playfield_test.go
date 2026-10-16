package config

import "testing"

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
