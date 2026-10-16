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
var Modes = []Mode{Stroll, LastTrain}

// LastTrain is Stroll plus trains on the track, the hop combo and the day
// and night cycle (DESIGN §6.2). Its own mechanics come in one level at a
// time (D27).
var LastTrain = Mode{
	ID:   "lasttrain",
	Name: "Last Train",
	Desc: "Stroll, plus trains on the line.",
	New: []Mechanic{
		{"Trains", "watch the lights, then cross"},
		{"Hop combo", "keep hopping forward for x2, x3, x4"},
		{"Day and night", "dawn to night as you fill the nooks"},
	},
	Also:   []string{"Road, river, diving platforms, herons"},
	Levels: lastTrainLevels(),
}

// lastTrainLevels are Stroll's levels with Last Train's mechanics: trains
// from level 1, the combo from level 2 and the day and night cycle from
// level 3, each with a hint on the level that brings it in (D27).
func lastTrainLevels() ModeLevels {
	m := StrollLevels
	m.Levels = append([]LevelDef(nil), m.Levels...)
	m.Levels[0].Hint = "WATCH THE CROSSING LIGHTS, THEN CROSS THE LINE"
	m.Levels[1].Hint = "KEEP HOPPING FORWARD FOR A COMBO: x2, x3, x4"
	m.Levels[2].Hint = "DAWN TO NIGHT: THE DARK FALLS AS YOU FILL THE NOOKS"
	m.Trains = true
	m.ComboFrom, m.CycleFrom = 2, 3
	return m
}

// ModesInProgress are modes being built and not yet in Modes. Debug
// builds can play them with -mode (D37).
var ModesInProgress = []Mode{RushHour}

// RushHour is Last Train plus the road in bad weather (DESIGN §6.3): rush
// hour waves from level 1, then fog and rain (M12). It keeps the trains,
// the combo and the day and night cycle from its level 1 (D27, D41).
var RushHour = Mode{
	ID:   "rushhour",
	Name: "Rush Hour",
	Desc: "Last Train, plus the road in bad weather.",
	New: []Mechanic{
		{"Rush hour", "a red light: a wave of traffic is coming"},
	},
	Also:   []string{"Trains, hop combo, day and night", "Road, river, diving platforms, herons"},
	Levels: rushHourLevels(),
}

// rushHourLevels are Last Train's levels with its mechanics from level 1,
// and Rush Hour's own coming in one a level (D41).
func rushHourLevels() ModeLevels {
	m := lastTrainLevels()
	m.Levels = append([]LevelDef(nil), m.Levels...)
	m.Levels[0].Hint = "A RED LIGHT ON A LANE: A RUSH OF TRAFFIC IS COMING"
	m.Levels[1].Hint, m.Levels[2].Hint = "", "" // Last Train's, already known
	m.ComboFrom, m.CycleFrom = 1, 1
	m.EventsFrom[Wave] = 1
	return m
}
