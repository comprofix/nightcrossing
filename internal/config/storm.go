package config

// Storm is Rush Hour plus the river in a storm (DESIGN §6.4): currents from
// level 1 and wind gusts from level 2. It keeps everything from the modes
// before it from its level 1 (D27, D45).
var Storm = Mode{
	ID:   "storm",
	Name: "Storm",
	Desc: "Rush Hour, plus the river in a storm.",
	New: []Mechanic{
		{"Currents", "ripples pull you along a platform"},
		{"Gusts", "the wind pushes you one cell"},
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

// stormLevels are Rush Hour's levels with all its events from level 1, and
// Storm's own coming in one a level (D50).
func stormLevels() ModeLevels {
	m := rushHourLevels()
	m.Levels = append([]LevelDef(nil), m.Levels...)
	for i := range m.Levels {
		m.Levels[i].Currents = stormCurrents[i]
	}
	m.Levels[0].Hint = "RIPPLES SHOW A CURRENT: IT PULLS YOU ALONG THE LOG"
	m.Levels[1].Hint = "GUSTS FROM THE WEST OR EAST PUSH YOU ONE CELL"
	m.Levels[2].Hint = "" // not Rush Hour's rain hint: it rains from level 1 here; floods come in M14
	m.EventsFrom[Wave], m.EventsFrom[Fog], m.EventsFrom[Rain] = 1, 1, 1
	m.EventsFrom[Gust] = 2
	return m
}
