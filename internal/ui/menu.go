// Package ui has the shared menu widget and drawing helpers for panels and
// text.
package ui

import (
	"github.com/comprofix/nightcrossing/internal/input"
)

// Item is one line of a Menu.
type Item struct {
	Label    string
	Value    func() string // optional: shown after the label, e.g. "On"
	Disabled bool
	// DisabledIf, if set, disables the item while it returns true, e.g. a
	// volume while its sound is switched off.
	DisabledIf func() bool
	// Slider, if set, makes the item a slider: a bar filled to its 0–1 value
	// and the value as a percentage, in place of Value.
	Slider  func() float64
	Confirm func()        // Confirm on this item
	Adjust  func(dir int) // Left (-1) or Right (+1) on this item
}

// IsDisabled reports whether the item can't be chosen right now.
func (it Item) IsDisabled() bool {
	return it.Disabled || (it.DisabledIf != nil && it.DisabledIf())
}

// Menu is a vertical list of items with a cursor. Up and Down move (with
// repeat, wrapping and skipping disabled items), Confirm activates, and
// Left/Right adjust values.
type Menu struct {
	Items  []Item
	Cursor int
	// OnMove and OnConfirm are called after the cursor moves or an item is
	// activated, for sounds.
	OnMove    func()
	OnConfirm func()
	// Rows, if set, is how many items show at once: a longer menu scrolls
	// to keep the cursor in view.
	Rows int

	top int // the first item shown, while it scrolls
}

// Update handles one tick of input.
func (m *Menu) Update(in *input.Input) {
	if len(m.Items) == 0 {
		return
	}
	defer func() { m.top = m.First() }()
	m.skipDisabled(1)
	switch {
	case in.Repeat(input.Up):
		m.move(-1)
	case in.Repeat(input.Down):
		m.move(1)
	case in.JustPressed(input.Confirm):
		it := m.Items[m.Cursor]
		if it.IsDisabled() {
			return
		}
		if it.Confirm != nil {
			it.Confirm()
		} else if it.Adjust != nil {
			it.Adjust(1)
		}
		if m.OnConfirm != nil {
			m.OnConfirm()
		}
	case in.Repeat(input.Left):
		m.adjust(-1)
	case in.Repeat(input.Right):
		m.adjust(1)
	}
}

func (m *Menu) move(dir int) {
	if m.step(dir) && m.OnMove != nil {
		m.OnMove()
	}
}

// step moves the cursor to the next enabled item in direction dir and
// reports whether it moved.
func (m *Menu) step(dir int) bool {
	start := m.Cursor
	for range m.Items {
		m.Cursor = (m.Cursor + dir + len(m.Items)) % len(m.Items)
		if !m.Items[m.Cursor].IsDisabled() {
			break
		}
	}
	return m.Cursor != start
}

func (m *Menu) adjust(dir int) {
	it := m.Items[m.Cursor]
	if it.IsDisabled() || it.Adjust == nil {
		return
	}
	it.Adjust(dir)
	if m.OnMove != nil {
		m.OnMove()
	}
}

// skipDisabled moves the cursor off a disabled item, if it is on one,
// silently.
func (m *Menu) skipDisabled(dir int) {
	m.Cursor = min(max(m.Cursor, 0), len(m.Items)-1)
	if m.Items[m.Cursor].IsDisabled() {
		m.step(dir)
	}
}

// Scrolls reports whether the menu has more items than it shows at once.
func (m *Menu) Scrolls() bool { return m.Rows > 0 && len(m.Items) > m.Rows }

// Shown is how many items show at once.
func (m *Menu) Shown() int {
	if !m.Scrolls() {
		return len(m.Items)
	}
	return m.Rows
}

// First is the first item shown: the top one, unless the menu scrolls;
// then the one that keeps the cursor in view, moving as little as it can
// from where it was.
func (m *Menu) First() int {
	if !m.Scrolls() {
		return 0
	}
	top := min(max(m.top, m.Cursor-m.Rows+1), m.Cursor)
	return min(max(top, 0), len(m.Items)-m.Rows)
}

// Text returns item i's text: the label, then its value if any. A slider's
// value is its percentage.
func (m *Menu) Text(i int) string {
	it := m.Items[i]
	if it.Slider != nil {
		return it.Label + ": " + percent(min(max(it.Slider(), 0), 1))
	}
	if it.Value == nil {
		return it.Label
	}
	return it.Label + ": " + it.Value()
}

// OnOff is a Value for a boolean setting.
func OnOff(v *bool) func() string {
	return func() string {
		if *v {
			return "On"
		}
		return "Off"
	}
}
