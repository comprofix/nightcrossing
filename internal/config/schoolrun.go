package config

// SchoolRun is Storm plus the escort (DESIGN §6.5, D53, D54): a line of
// toadlets follows the hero's trail across: one toadlet, on every level.
// It keeps everything from Storm, storms and all, from its level 1 (D27).
var SchoolRun = Mode{
	ID:   "schoolrun",
	Name: "School Run",
	Desc: "Storm, plus toadlets to walk across.",
	New: []Mechanic{
		{"Escort", "a toadlet copies your hops, one behind"},
		{"Gather up", "stop on the grass and it catches up"},
		{"Home together", "+50 for each toadlet home"},
	},
	Also:   []string{"Everything in Stroll, Last Train, Rush Hour and Storm", PickupsAlso},
	Levels: schoolRunLevels(),
	Bonus:  SchoolGates,
}

// BunchTicks is how long the hero stands still on a safe row before the
// toadlets behind it join it in its cell (D53). A starting value.
const BunchTicks = TPS / 2

// schoolRunToadlets are the toadlets on every level: one, as a second on
// top of predators and emergency vehicles was too much (D54).
const schoolRunToadlets = 1

// schoolRunLevels are Storm's levels with all of Storm from level 1, and
// a toadlet to escort, with hints for the line and bunching up (D54).
func schoolRunLevels() ModeLevels {
	m := stormLevels()
	m.Levels = append([]LevelDef(nil), m.Levels...)
	for i := range m.Levels {
		m.Levels[i].Toadlets = schoolRunToadlets
		m.Levels[i].Hint = ""
	}
	m.Levels[0].Hint = "YOUR TOADLET COPIES YOUR HOPS: KEEP IT SAFE"
	m.Levels[1].Hint = "STOP ON THE VERGE AND YOUR TOADLET CATCHES UP"
	m.Levels[MixedFrom-1].Hint = MixedHint
	m.EventsFrom[Rain] = 1
	m.WindFrom, m.FloodFrom = 1, 1
	return m
}

// EscortCheckMins is the bot check's limit: the autopilot, escorting, must
// clear each School Run level within this many minutes (D55). A starting
// value.
const EscortCheckMins = 5
