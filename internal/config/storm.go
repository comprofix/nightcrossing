package config

// Storm is Rush Hour plus storms that roll in and away over the river and
// road (DESIGN §6.4, D51): currents and wet storms from level 1, dry
// storms and plain rain from level 2 and floods in wet storms from level
// 3, under a storm sky. It keeps everything from the modes before it from
// its level 1 (D27, D45).
var Storm = Mode{
	ID:   "storm",
	Name: "Storm",
	Desc: "Rush Hour, plus storms that roll in and away.",
	New: []Mechanic{
		{"Currents", "ripples pull you along a platform"},
		{"Wet storms", "rain, lightning, and floods on the verge"},
		{"Dry storms", "high winds push you one cell"},
	},
	Also:   []string{"Everything in Stroll, Last Train and Rush Hour"},
	Levels: stormLevels(),
}

// stormCurrents are each level's current rows (D50): one on levels 1–3,
// two on 4–6, three on 7–8, strength 3 rising to 6. Only rows of logs and
// ships have them, never the lily pads, and the first is against the flow
// on a row of long logs. Starting values, tuned by play.
var stormCurrents = [][]Current{
	{{Row: 3, Push: -3}},
	{{Row: 3, Push: -3}},
	{{Row: 1, Push: 4}},
	{{Row: 1, Push: 4}, {Row: 5, Push: -4}},
	{{Row: 1, Push: -5}, {Row: 3, Push: 5}},
	{{Row: 1, Push: 5}, {Row: 5, Push: -5}},
	{{Row: 1, Push: 6}, {Row: 3, Push: -6}, {Row: 5, Push: 6}},
	{{Row: 1, Push: -6}, {Row: 3, Push: 6}, {Row: 5, Push: -6}},
}

// stormLevels are Rush Hour's levels with waves and fog from level 1, plain
// rain from level 2 and less often, the storm sky throughout, and Storm's
// own mechanics coming in one a level (D45, D49–D51).
func stormLevels() ModeLevels {
	m := rushHourLevels()
	m.Levels = append([]LevelDef(nil), m.Levels...)
	for i := range m.Levels {
		m.Levels[i].Currents = stormCurrents[i]
	}
	m.Levels[0].Hint = "RIPPLES SHOW A CURRENT: IT PULLS YOU ALONG THE LOG"
	m.Levels[1].Hint = "HIGH WINDS PUSH YOU ONE CELL: STAND WHERE IT'S SAFE"
	m.Levels[2].Hint = "A WET STORM FLOODS THE VERGE: THE RIVER RUNS FASTER"
	m.EventsFrom[Wave], m.EventsFrom[Fog], m.EventsFrom[Rain], m.EventsFrom[StormFront] = 1, 1, 2, 1
	m.QuietPct[Rain] = 150
	m.WindFrom, m.FloodFrom = 2, 3
	m.StormFrom = 1
	return m
}
