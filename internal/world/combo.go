package world

import "github.com/comprofix/nightcrossing/internal/config"

// Combo is the hop combo, on levels that have it (DESIGN §6.2, D38).
type Combo struct {
	Hops int // forward hops into new rows without a break; 0 for no combo
	Left int // ticks left before the combo ends without another one
}

// Mult is the combo's score multiplier.
func (c Combo) Mult() int { return config.ComboMult(c.Hops) }

// countHop counts a hop into a new row into the combo, on a level with
// the combo, and restarts its clock. A hop that raises the multiplier
// gives out ComboUp.
func (w *World) countHop() {
	if !w.Level.Combo {
		return
	}
	before := w.Combo.Mult()
	w.Combo.Hops++
	w.Combo.Left = config.ComboTicks
	if w.Combo.Mult() > before {
		w.emit(ComboUp)
	}
}

// comboPoints scores base points times the combo's multiplier. What the
// multiplier added goes to Stats.Combo, and anything multiplied to Gained,
// for the pop-ups.
func (w *World) comboPoints(base int) int {
	pts := base * w.Combo.Mult()
	if pts > base {
		w.Stats.Combo += pts - base
		w.Gained += pts
	}
	return pts
}

// updateCombo runs the combo's clock down, and ends the combo when it runs
// out. Riding a platform pauses it, so waiting on the river does not break
// a combo (D38).
func (w *World) updateCombo() {
	if w.Combo.Hops == 0 {
		return
	}
	if p := &w.Player; !p.Hopping() && w.Lanes[p.Row].Kind == config.River {
		return
	}
	if w.Combo.Left--; w.Combo.Left <= 0 {
		w.Combo = Combo{}
	}
}
