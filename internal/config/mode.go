package config

// Mechanic is one thing a mode brings, as its intro screen lists it
// (DESIGN §7, D25): a name and a short line on how it plays.
type Mechanic struct {
	Name string
	Desc string
}

// Mode is what the player picks before a game (DESIGN §5.6, D23). Each mode
// is the one before it plus new mechanics; every mode uses the same base
// numbers (D13), so a mode differs only in its mechanics and levels.
type Mode struct {
	ID     string     // stable key, used in saved files
	Name   string     // shown in menus
	Desc   string     // one line, under the MODE menu and on the intro
	New    []Mechanic // what this mode brings, for the intro
	Also   []string   // what carries over from the modes before it
	Levels ModeLevels // its levels and loops
}

// Stroll is the base game, the first mode (DESIGN §5.6).
var Stroll = Mode{
	ID:   "stroll",
	Name: "Stroll",
	Desc: "Road, verge and river: get five toads home.",
	New: []Mechanic{
		{"Road", "cars, vans, bikes and trucks"},
		{"River", "ride logs, barges and the ferry"},
		{"Lily pads", "they dive when they flash"},
		{"Sinking ships", "run to the stern or hop off"},
		{"Herons", "don't land on one"},
	},
	Levels: StrollLevels,
}

// Modes are the built modes, in order: only built modes are shown, and the
// list grows with each version (D26).
var Modes = []Mode{Stroll}
