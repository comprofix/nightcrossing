package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Lane is one row of the playfield as it runs.
//
// Each lane is a loop: positions run from 0 to Loop and wrap. A position u
// is drawn at x = u - Buffer from the field's left edge. Buffer is at least
// the longest vehicle and Loop at least the field's width plus Buffer, so a
// vehicle wraps only while it is wholly off the field and never pops in
// (PLAN §3).
type Lane struct {
	Kind     config.LaneKind
	Dir      int // +1 right, -1 left
	Speed    int // SubPx units per tick, after the skill's speed
	Loop     int // SubPx units
	Vehicles []Vehicle
}

// Vehicle is one vehicle on a road lane.
type Vehicle struct {
	Kind config.Vehicle
	U    int // its left end's position along the loop, in SubPx units
}

// Buffer is how far off the field's left edge a lane's loop begins, in
// SubPx units: the longest vehicle.
const Buffer = config.LongestVehicle * config.SubPx

// fieldWidth is the playfield's width in SubPx units.
const fieldWidth = config.Cols * config.Cell * config.SubPx

// newLane lays out a level's lane for a skill. The pattern repeats until
// it fills the loop, and the whole lane starts at a random point along it,
// so each game looks a little different but the pattern holds.
func newLane(l config.Lane, skill config.Skill, rng *rand.Rand) Lane {
	lane := Lane{Kind: l.Kind, Dir: l.Dir}
	if l.Kind == config.Safe || len(l.Pattern) == 0 {
		return lane
	}
	lane.Speed = max(1, l.Speed*skill.SpeedPct/100)

	pattern := 0
	for _, s := range l.Pattern {
		pattern += (s.Vehicle.Length + s.Gap) * config.SubPx
	}
	repeats := (fieldWidth + Buffer + pattern - 1) / pattern
	lane.Loop = repeats * pattern

	u := rng.IntN(lane.Loop)
	for range repeats {
		for _, s := range l.Pattern {
			lane.Vehicles = append(lane.Vehicles, Vehicle{Kind: s.Vehicle, U: u % lane.Loop})
			u += (s.Vehicle.Length + s.Gap) * config.SubPx
		}
	}
	return lane
}

// update moves the lane's vehicles one tick.
func (l *Lane) update() {
	for i := range l.Vehicles {
		v := &l.Vehicles[i]
		v.U = ((v.U+l.Dir*l.Speed)%l.Loop + l.Loop) % l.Loop
	}
}

// X is where v's left end is, from the field's left edge, in SubPx units.
func (v Vehicle) X() int { return v.U - Buffer }

// span is the part of the field v covers, left to right, in SubPx units,
// trimmed by margin pixels at each end.
func (v Vehicle) span(margin int) (left, right int) {
	m := margin * config.SubPx
	x := v.X()
	return x + m, x + v.Kind.Length*config.SubPx - m
}

// hits reports whether any vehicle in the lane overlaps left–right,
// with each vehicle's ends trimmed by margin pixels.
func (l *Lane) hits(left, right, margin int) bool {
	for _, v := range l.Vehicles {
		vl, vr := v.span(margin)
		if vl < right && left < vr {
			return true
		}
	}
	return false
}
