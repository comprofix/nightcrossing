package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/input"
)

func feed(steps ...input.Step) *input.Input {
	return input.New(&input.Script{Steps: steps})
}

func TestMenuMovesAndSkipsDisabled(t *testing.T) {
	m := &Menu{Items: []Item{{Label: "A"}, {Label: "B", Disabled: true}, {Label: "C"}}}
	moves := 0
	m.OnMove = func() { moves++ }
	in := feed(input.Press(ebiten.KeyArrowDown), input.Idle(1), input.Press(ebiten.KeyArrowDown), input.Idle(1), input.Press(ebiten.KeyArrowUp))
	want := []int{2, 2, 0, 0, 2} // B is skipped; the menu wraps both ways
	for i, w := range want {
		in.Update()
		m.Update(in)
		if m.Cursor != w {
			t.Fatalf("tick %d: cursor %d, want %d", i, m.Cursor, w)
		}
	}
	if moves != 3 {
		t.Errorf("OnMove called %d times, want 3", moves)
	}
}

func TestMenuStartsOffADisabledItem(t *testing.T) {
	m := &Menu{Items: []Item{{Label: "A", Disabled: true}, {Label: "B"}}}
	in := feed(input.Idle(1))
	in.Update()
	m.Update(in)
	if m.Cursor != 1 {
		t.Errorf("cursor %d, want 1", m.Cursor)
	}
}

func TestMenuConfirmAndAdjust(t *testing.T) {
	confirmed, level := 0, 5
	on := false
	m := &Menu{Items: []Item{
		{Label: "Go", Confirm: func() { confirmed++ }},
		{Label: "Level", Value: func() string { return string(rune('0' + level)) }, Adjust: func(d int) { level += d }},
		{Label: "Flag", Value: OnOff(&on), Adjust: func(int) { on = !on }},
	}}
	in := feed(
		input.Press(ebiten.KeyEnter), input.Idle(1),
		input.Press(ebiten.KeyArrowDown), input.Idle(1),
		input.Press(ebiten.KeyArrowRight), input.Idle(1),
		input.Press(ebiten.KeyArrowRight), input.Idle(1),
		input.Press(ebiten.KeyArrowLeft), input.Idle(1),
		input.Press(ebiten.KeyArrowDown), input.Idle(1),
		input.Press(ebiten.KeyEnter), // Confirm on an Adjust-only item adjusts it
	)
	for range 13 {
		in.Update()
		m.Update(in)
	}
	if confirmed != 1 {
		t.Errorf("confirmed %d times, want 1", confirmed)
	}
	if level != 6 {
		t.Errorf("level %d, want 6", level)
	}
	if !on || m.Text(2) != "Flag: On" {
		t.Errorf("flag %v, text %q", on, m.Text(2))
	}
	if m.Text(0) != "Go" {
		t.Errorf("text %q", m.Text(0))
	}
}

func TestMenuIgnoresDisabledConfirm(t *testing.T) {
	called := false
	m := &Menu{Items: []Item{{Label: "X", Disabled: true, Confirm: func() { called = true }}}}
	in := feed(input.Press(ebiten.KeyEnter))
	in.Update()
	m.Update(in)
	if called {
		t.Error("a disabled item was activated")
	}
}

func TestBlink(t *testing.T) {
	if !Blink(0, 40) || !Blink(19, 40) || Blink(20, 40) || Blink(39, 40) || !Blink(40, 40) {
		t.Error("Blink should be on for the first half of each period")
	}
}

func TestSkippingADisabledItemIsSilent(t *testing.T) {
	m := &Menu{Items: []Item{{Label: "A", Disabled: true}, {Label: "B"}}}
	m.OnMove = func() { t.Error("OnMove called when skipping a disabled first item") }
	in := feed(input.Idle(1))
	in.Update()
	m.Update(in)
}

func TestDisabledIfIsSkippedAndSliderText(t *testing.T) {
	on, v := false, 0.6
	m := &Menu{Items: []Item{
		{Label: "Music", Adjust: func(int) { on = !on }},
		{Label: "Music volume", Slider: func() float64 { return v }, DisabledIf: func() bool { return !on },
			Adjust: func(d int) { v += 0.1 * float64(d) }},
		{Label: "Back"},
	}}
	in := feed(input.Press(ebiten.KeyArrowDown), input.Idle(1), input.Press(ebiten.KeyArrowUp), input.Idle(1),
		input.Press(ebiten.KeyEnter), input.Idle(1), input.Press(ebiten.KeyArrowDown))
	want := []int{2, 2, 0, 0, 0, 0, 1} // off: the slider is skipped; on: it is not
	for i, w := range want {
		in.Update()
		m.Update(in)
		if m.Cursor != w {
			t.Fatalf("tick %d: cursor %d, want %d", i, m.Cursor, w)
		}
	}
	if got := m.Text(1); got != "Music volume: 60%" {
		t.Errorf("slider text %q", got)
	}
}

// TestMenuScrolls: a menu with more items than Rows shows Rows at a time,
// moving the window only as far as it must to keep the cursor in view,
// both ways and when the cursor wraps.
func TestMenuScrolls(t *testing.T) {
	m := &Menu{Rows: 3}
	for range 6 {
		m.Items = append(m.Items, Item{Label: "X"})
	}
	var steps []input.Step
	for _, k := range []ebiten.Key{
		ebiten.KeyArrowDown, ebiten.KeyArrowDown, ebiten.KeyArrowDown, ebiten.KeyArrowDown, // to 4
		ebiten.KeyArrowUp, ebiten.KeyArrowUp, ebiten.KeyArrowUp, // back to 1
		ebiten.KeyArrowUp, ebiten.KeyArrowUp, // to 0, then round to 5
	} {
		steps = append(steps, input.Press(k), input.Idle(1))
	}
	in := feed(steps...)
	want := []struct{ cursor, first int }{
		{1, 0}, {2, 0}, {3, 1}, {4, 2},
		{3, 2}, {2, 2}, {1, 1},
		{0, 0}, {5, 3},
	}
	for i, w := range want {
		for range 2 {
			in.Update()
			m.Update(in)
		}
		if m.Cursor != w.cursor || m.First() != w.first {
			t.Fatalf("move %d: cursor %d, first %d; want %d, %d", i, m.Cursor, m.First(), w.cursor, w.first)
		}
	}
	if !m.Scrolls() || m.Shown() != 3 {
		t.Errorf("scrolls %v, shows %d: want 3 of 6", m.Scrolls(), m.Shown())
	}
	short := &Menu{Rows: 8, Items: m.Items}
	if short.Scrolls() || short.Shown() != 6 || short.First() != 0 {
		t.Errorf("a menu shorter than Rows: scrolls %v, shows %d", short.Scrolls(), short.Shown())
	}
}
