package config

// SchoolRun is Storm plus the escort (DESIGN §6.5, D53, D54): a line of
// toadlets follows the hero's trail across, one toadlet on levels 1–4 and
// two on 5–8 and the loops. It keeps everything from Storm, storms and
// all, from its level 1 (D27).
var SchoolRun = Mode{
	ID:   "schoolrun",
	Name: "School Run",
	Desc: "Storm, plus toadlets to walk across.",
	New: []Mechanic{
		{"Escort", "a toadlet copies your hops, one behind"},
		{"More toadlets", "a second from level 5"},
		{"Home together", "+50 for each toadlet home"},
	},
	Also:   []string{"Everything in Stroll, Last Train, Rush Hour and Storm"},
	Levels: schoolRunLevels(),
}

// BunchTicks is how long the hero stands still on a safe row before the
// toadlets behind it join it in its cell (D53). A starting value.
const BunchTicks = TPS / 2

// schoolRunToadlets are each level's toadlets: never more than two (D54).
var schoolRunToadlets = []int{1, 1, 1, 1, 2, 2, 2, 2}

// schoolRunLevels are Storm's levels with all of Storm from level 1, and
// the line growing over the levels, with hints for the line, bunching up
// and the second toadlet (D54).
func schoolRunLevels() ModeLevels {
	m := stormLevels()
	m.Levels = append([]LevelDef(nil), m.Levels...)
	for i := range m.Levels {
		m.Levels[i].Toadlets = schoolRunToadlets[i]
		m.Levels[i].Hint = ""
	}
	m.Levels[0].Hint = "YOUR TOADLET COPIES YOUR HOPS: KEEP IT SAFE"
	m.Levels[1].Hint = "STOP ON THE VERGE AND YOUR TOADLET CATCHES UP"
	m.Levels[4].Hint = "TWO TOADLETS NOW: THE SECOND COPIES THE FIRST"
	m.EventsFrom[Rain] = 1
	m.WindFrom, m.FloodFrom = 1, 1
	return m
}

// EscortCheckMins is the bot check's limit: the autopilot, escorting, must
// clear each School Run level within this many minutes (D55). A starting
// value.
const EscortCheckMins = 5

// ToadletPx is a toadlet's size, about two thirds of the hero's (D56).
const ToadletPx = 16
