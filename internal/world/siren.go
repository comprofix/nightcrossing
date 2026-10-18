package world

import (
	"slices"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Siren is an emergency vehicle's coming (DESIGN §6.9, D71–D73): its
// lights flash at the edge of the emergency lane while the lane empties
// and the lanes either side bunch up and stand; it comes through the
// lane; and the traffic goes back to its places.
type Siren struct {
	Phase   SirenPhase
	Row     int // the emergency lane
	Vehicle int // which of config.Responders comes
	X       int // the vehicle's left end through its run, in SubPx units from the field's left edge
	T       int // ticks into its phase

	clearFor int // ticks the lane has been clear with the trap not yet closed
}

// SirenPhase is where an emergency vehicle's coming is.
type SirenPhase int

// The phases.
const (
	SirenAway    SirenPhase = iota
	SirenWarning            // lights at the lane's edge: the lane empties and the lanes either side bunch up
	SirenRun                // the vehicle comes through: deadly in its lane
	SirenPassed             // it has gone; the queues stand, brake lights on
	SirenMoveOff            // brake lights off: the queues are about to move off
	SirenReturn             // the queues have moved off; the emergency lane is still empty a moment
)

// Responder is the vehicle coming.
func (s Siren) Responder() config.Responder { return config.Responders()[s.Vehicle] }

// Lights reports whether the vehicle's lights flash: through the warning
// and its run.
func (s Siren) Lights() bool { return s.Phase == SirenWarning || s.Phase == SirenRun }

// queues are the emergency lane's neighbours, the rows that bunch up.
func (s Siren) queues() [2]int { return [2]int{s.Row - 1, s.Row + 1} }

// warnSiren sends an emergency vehicle down one of config.EmergencyRows,
// if one is a road lane with road either side, and reports whether it did
// (D72). The lane's cars drive off, faster if they need to, and none come
// on; the lanes either side start to bunch up.
func (w *World) warnSiren() bool {
	var rows []int
	for _, r := range config.EmergencyRows {
		if w.Lanes[r].Kind == config.Road && w.Lanes[r-1].Kind == config.Road && w.Lanes[r+1].Kind == config.Road {
			rows = append(rows, r)
		}
	}
	n := config.RespondersOn(w.Level.Number)
	if len(rows) == 0 || n == 0 {
		return false
	}
	row := rows[w.rng.IntN(len(rows))]
	w.startSiren(row, w.rng.IntN(n))
	return true
}

// startSiren sends vehicle, one of config.Responders, down road row row.
func (w *World) startSiren(row, vehicle int) {
	w.Siren = Siren{Phase: SirenWarning, Row: row, Vehicle: vehicle}
	w.hold(row, Clear)
	for _, r := range w.Siren.queues() {
		w.hold(r, Queue)
	}
	w.emit(SirenSounds)
}

// hold gives road row r its role in the emergency. Each car's place in
// the lane's pattern is kept, to go back to after (unless the lane is
// still finding its way back from the last one), and the cars wholly off
// the field are away: none come on.
func (w *World) hold(r int, role Role) {
	l := &w.Lanes[r]
	if !l.held {
		for i := range l.Entities {
			l.Entities[i].Home = l.Entities[i].U
		}
	}
	l.held, l.Role, l.Brakes = true, role, role == Queue
	for i := range l.Entities {
		if e := &l.Entities[i]; !e.Out && e.off() {
			e.Away = true
		}
	}
	entry, exit := l.edges()
	speed := l.Speed + l.Speed*l.Surge/100
	switch role {
	case Clear: // off the field by config.CloseTicks, at no less than config.ClearPct
		far := exit
		for _, e := range l.Entities {
			if !e.Hidden() {
				_, back := l.ends(e)
				far = min(far, back)
			}
		}
		l.pace = max(speed*config.ClearPct/100, ceilDiv(exit-far, config.CloseTicks))
	case Queue: // filling the field in about config.CloseTicks, the extra cars coming on for the last part of it
		free := exit - entry
		for _, e := range l.Entities {
			if !e.Hidden() {
				front, back := l.ends(e)
				free -= min(front, exit) - max(back, entry) + config.QueueGap*config.SubPx
			}
		}
		l.pace = max(1, ceilDiv(max(free, 0), config.CloseTicks-config.ExtrasTicks))
	}
}

func ceilDiv(a, b int) int { return (a + b - 1) / b }

// updateSiren runs the emergency vehicle's coming one tick on (D72), and
// when it is over sends spell s back to its quiet.
func (w *World) updateSiren(s *Spell) {
	sr := &w.Siren
	sr.T++
	switch sr.Phase {
	case SirenWarning:
		if sr.T < config.EscapeTicks || !w.laneClear(sr.Row) {
			return
		}
		if !w.shut(sr.queues()[0]) || !w.shut(sr.queues()[1]) {
			if sr.clearFor++; sr.clearFor < config.RunWaitTicks {
				return
			}
		}
		v := sr.Responder()
		sr.Phase, sr.T, sr.X = SirenRun, 0, -v.Length*config.SubPx
		if w.Lanes[sr.Row].Dir < 0 {
			sr.X = fieldWidth
		}
		s.Phase = SpellOn
		w.emit(SirenComing)
	case SirenRun:
		v := sr.Responder()
		sr.X += w.Lanes[sr.Row].Dir * v.Speed
		if sr.X >= fieldWidth || sr.X+v.Length*config.SubPx <= 0 {
			sr.Phase, sr.T = SirenPassed, 0
			w.emit(SirenGone)
		}
	case SirenPassed:
		if sr.T >= config.ReleaseTicks {
			sr.Phase, sr.T = SirenMoveOff, 0
			for _, r := range sr.queues() {
				w.Lanes[r].Brakes = false
			}
			w.emit(BrakesOff)
		}
	case SirenMoveOff:
		if sr.T >= config.MoveOffTicks {
			sr.Phase, sr.T = SirenReturn, 0
			for _, r := range sr.queues() {
				w.Lanes[r].Role = Unheld
			}
			w.emit(TrafficMoves)
		}
	case SirenReturn:
		if sr.T >= config.ReturnTicks {
			w.Lanes[sr.Row].Role = Unheld
			*sr = Siren{}
			s.Phase, s.Left = SpellQuiet, s.quiet(w.rng, false)
		}
	}
}

// laneClear reports whether no car of road row r is on the field.
func (w *World) laneClear(r int) bool {
	for _, e := range w.Lanes[r].Entities {
		if !e.Hidden() && !e.off() {
			return false
		}
	}
	return true
}

// shut reports whether road row r's queue is closed: every car standing,
// and no room at the field's edge for another to come on.
func (w *World) shut(r int) bool {
	l := &w.Lanes[r]
	for _, e := range l.Entities {
		if !e.Hidden() && !e.Halted {
			return false
		}
	}
	return !l.room()
}

// moveHeld moves the cars of each lane held for an emergency vehicle one
// tick on, its role's way (D72): an emergency lane's drive off; a queue's
// bunch up and stand, giving way to the hero and toadlets; and once the
// queues move off, and the emergency lane's traffic comes back, each car
// goes back to its place.
func (w *World) moveHeld() {
	for r := range w.Lanes {
		l := &w.Lanes[r]
		if !l.held {
			continue
		}
		switch l.Role {
		case Clear:
			l.driveOff(w.bodiesIn(r))
		case Queue:
			l.bunch(w.bodiesIn(r), w.Siren.Phase != SirenWarning || w.Siren.T >= config.ExtrasTicks)
		default:
			l.held = !l.goBack()
		}
	}
}

// edges are where the lane's traffic comes onto the field and leaves it,
// along the flow (see ends).
func (l *Lane) edges() (entry, exit int) {
	if l.Dir > 0 {
		return 0, fieldWidth
	}
	return -fieldWidth, 0
}

// ends are e's front, the end it heads towards, and its back, in SubPx
// units along the lane's flow: they grow downstream.
func (l *Lane) ends(e Entity) (front, back int) {
	x, n := e.X(), e.Kind.Length*config.SubPx
	if l.Dir > 0 {
		return x + n, x
	}
	return -x, -x - n
}

// setFront puts e with its front at front, along the flow.
func (l *Lane) setFront(e *Entity, front int) {
	x := front - e.Kind.Length*config.SubPx
	if l.Dir < 0 {
		x = -front
	}
	e.U = l.wrap(x + Buffer)
}

// driveOff moves an emergency lane's cars off the field at its pace;
// each is away once off it, and none comes on. A car with the hero or a
// toadlet ahead of it keeps to the lane's own speed, so it never comes at
// them faster than the traffic would (D72). bodies are as for bunch.
func (l *Lane) driveOff(bodies [][2]int) {
	for i := range l.Entities {
		e := &l.Entities[i]
		switch {
		case e.Out:
			e.U = e.Home
		case e.Away:
		case e.off():
			e.Away = true
		default:
			front, back := l.ends(*e)
			m := l.pace
			for _, b := range bodies {
				if b[1] > back {
					m = min(m, l.Speed+l.Speed*l.Surge/100)
				}
			}
			l.setFront(e, front+m)
		}
	}
}

// bunch moves a queue's cars one tick on: each closes on the one ahead at
// the lane's pace until config.QueueGap behind it, the first stopping at
// the field's edge it leaves by, and none comes nearer than that to a
// body in the lane ahead of it, so the queue leaves it a pocket (D72).
// Cars already leaving the field go on and are away once off it, and new
// ones come on behind, once extras is true, until the queue reaches the
// edge they come in by.
// bodies are the hero and toadlets in the lane, near and far ends along
// the flow.
func (l *Lane) bunch(bodies [][2]int, extras bool) {
	_, exit := l.edges()
	gap := config.QueueGap * config.SubPx
	var cars []int
	for i := range l.Entities {
		if e := &l.Entities[i]; e.Out {
			e.U = e.Home
		} else if !e.Away {
			cars = append(cars, i)
		}
	}
	slices.SortFunc(cars, func(a, b int) int {
		fa, _ := l.ends(l.Entities[a])
		fb, _ := l.ends(l.Entities[b])
		return fb - fa
	})
	ahead := 1 << 40 // the back of the car ahead
	for _, i := range cars {
		e := &l.Entities[i]
		front, back := l.ends(*e)
		m := min(l.pace, ahead-gap-front)
		if front <= exit {
			m = min(m, exit-front)
		}
		for _, b := range bodies {
			if b[1] > back {
				m = min(m, b[0]-gap-front)
			}
		}
		m = max(m, 0)
		l.setFront(e, front+m)
		e.Halted = m == 0
		ahead = back + m
		if ahead >= exit {
			e.Away = true
		}
	}
	if extras && l.room() {
		l.bringOn()
	}
}

// room reports whether a queue has room for another car to come on behind
// it, at the field's edge its traffic comes in by.
func (l *Lane) room() bool {
	entry, _ := l.edges()
	last := 1 << 40
	for _, e := range l.Entities {
		if !e.Hidden() {
			_, back := l.ends(e)
			last = min(last, back)
		}
	}
	return last-config.QueueGap*config.SubPx >= entry
}

// bringOn brings a car on to close a queue, just off the field where its
// traffic comes in, taking the lane's own kinds in turn.
func (l *Lane) bringOn() {
	var kinds []config.Entity
	for _, e := range l.Entities {
		if !e.Wave && !e.Extra {
			kinds = append(kinds, e.Kind)
		}
	}
	if len(kinds) == 0 {
		return
	}
	entry, _ := l.edges()
	e := Entity{Kind: kinds[l.extras%len(kinds)], Extra: true}
	l.extras++
	l.setFront(&e, entry)
	e.Home = e.U
	l.Entities = append(l.Entities, e)
}

// goBack moves the cars of a lane no longer held back towards their
// places, at the lane's speed, and reports whether every one is there
// (D72). A car still out of place drives on with the rest; once off the
// field it takes its place if that is off the field too, and clear of the
// cars still out of place, or is away until it is. A queue's extra cars
// go once off the field.
func (l *Lane) goBack() bool {
	step := l.Step()
	kept := l.Entities[:0]
	for _, e := range l.Entities {
		if e.Extra && e.off() {
			continue
		}
		kept = append(kept, e)
	}
	l.Entities = kept
	done := true
	for i := range l.Entities {
		e := &l.Entities[i]
		if e.Out {
			e.U = e.Home
			continue
		}
		if !e.Away {
			e.U = l.wrap(e.U + step)
		}
		if e.U == e.Home && !e.Away && !e.Extra {
			continue
		}
		done = false
		if !e.Away && (!e.off() || e.Extra) {
			continue
		}
		e.Away = !l.homeFree(i)
		if !e.Away {
			e.U = e.Home
		}
	}
	return done
}

// homeFree reports whether car i's place is off the field and clear, by
// config.QueueGap, of every other car on the lane.
func (l *Lane) homeFree(i int) bool {
	e := l.Entities[i]
	at := e
	at.U = e.Home
	if !at.off() {
		return false
	}
	gap := config.QueueGap * config.SubPx
	n := e.Kind.Length * config.SubPx
	for j, o := range l.Entities {
		if j == i || o.Hidden() {
			continue
		}
		d := l.wrap(o.U - e.Home)
		if d < n+gap || d > l.Loop-o.Kind.Length*config.SubPx-gap {
			return false
		}
	}
	return true
}

// bodiesIn are the hero and toadlets in road row r, or landing in it,
// each as its near and far ends along r's flow: the queue gives way to
// them (D72).
func (w *World) bodiesIn(r int) [][2]int {
	l := &w.Lanes[r]
	var out [][2]int
	for i := Hero; i < len(w.Line); i++ {
		p := w.body(i)
		if i != Hero && w.Line[i].Riding || p.LaneRow() != r && p.Row != r {
			continue
		}
		lo, hi := min(p.X, p.toX), max(p.X, p.toX)+cellUnits
		if l.Dir > 0 {
			out = append(out, [2]int{lo, hi})
		} else {
			out = append(out, [2]int{-hi, -lo})
		}
	}
	return out
}

// sirenHits reports whether the emergency vehicle, coming through road row
// r, overlaps left–right, in SubPx units, trimmed by config.HitMargin at
// each end (D72).
func (w *World) sirenHits(r, left, right int) bool {
	s := w.Siren
	if s.Phase != SirenRun || s.Row != r {
		return false
	}
	m := config.HitMargin * config.SubPx
	return s.X+m < right && left < s.X+s.Responder().Length*config.SubPx-m
}

// Trapped reports whether body i is in the emergency lane, or landing in
// it, while the vehicle is coming (D72).
func (w *World) Trapped(i int) bool {
	s := w.Siren
	p := w.body(i)
	return s.Lights() && (p.LaneRow() == s.Row || p.Row == s.Row)
}

// trapped reports whether the hero or a toadlet out on the field is
// Trapped: the autopilot gets them out.
func (w *World) trapped() bool {
	for i := Hero; i < len(w.Line); i++ {
		if (i == Hero || !w.Line[i].Riding) && w.Trapped(i) {
			return true
		}
	}
	return false
}
