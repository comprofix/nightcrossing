package config

// Stroll's road: the same four lanes on every layout (DESIGN §3.2, D32).
// The first numbers, from M2.
var strollRoad = [Rows]Lane{
	8: {Kind: Road, Dir: -1, Speed: 10, Pattern: []Spawn{
		{Entity: Truck, Gap: 150}, {Entity: Truck, Gap: 230},
	}},
	9: {Kind: Road, Dir: 1, Speed: 16, Pattern: []Spawn{
		{Entity: CarRed, Gap: 90}, {Entity: CarBlue, Gap: 90}, {Entity: CarYellow, Gap: 170},
	}},
	10: {Kind: Road, Dir: -1, Speed: 28, Pattern: []Spawn{
		{Entity: Bike, Gap: 260},
	}},
	11: {Kind: Road, Dir: 1, Speed: 12, Pattern: []Spawn{
		{Entity: Van, Gap: 110}, {Entity: Van, Gap: 190},
	}},
}

// strollLayout is a Stroll layout: the far bank, the river rows given,
// the verge, the track as scenery, the road and the start (DESIGN §3.2).
func strollLayout(name string, river [5]Lane) *Layout {
	l := &Layout{Name: name, Lanes: strollRoad}
	l.Lanes[GoalRow] = Lane{Kind: Bank}
	for i, rl := range river {
		l.Lanes[1+i] = rl
	}
	l.Lanes[7] = Lane{Kind: Rail}
	return l
}

// River rows, used by the layouts below. The rows alternate direction, so
// the platforms in rows next to each other pass each other.
var (
	logsRight = Lane{Kind: River, Dir: 1, Speed: 10, Pattern: []Spawn{
		{Entity: LogLong, Gap: 96}, {Entity: LogShort, Gap: 120},
	}}
	logsLeft = Lane{Kind: River, Dir: -1, Speed: 12, Pattern: []Spawn{
		{Entity: LogShort, Gap: 96}, {Entity: LogLong, Gap: 120},
	}}
	logsSlow = Lane{Kind: River, Dir: 1, Speed: 8, Pattern: []Spawn{
		{Entity: LogLong, Gap: 72}, {Entity: LogLong, Gap: 144},
	}}
	logsQuick = Lane{Kind: River, Dir: -1, Speed: 14, Pattern: []Spawn{
		{Entity: LogShort, Gap: 84}, {Entity: LogShort, Gap: 120},
	}}
	logsShort = Lane{Kind: River, Dir: 1, Speed: 12, Pattern: []Spawn{
		{Entity: LogShort, Gap: 96}, {Entity: LogShort, Gap: 72}, {Entity: LogLong, Gap: 120},
	}}
)

// pads is a row of lily pad clumps moving left; dives marks the clumps
// that dive.
func pads(speed int, dives ...bool) Lane {
	clumps := []Spawn{{Entity: Pads3, Gap: 72}, {Entity: Pads3, Gap: 72}, {Entity: Pads2, Gap: 96}}
	for i, d := range dives {
		clumps[i].Dives = d
	}
	return Lane{Kind: River, Dir: -1, Speed: speed, Pattern: clumps}
}

// ships is the row of the ferry and a barge; sinks marks the barge as one
// that sinks. The ferry never does, so the row keeps a ship that stays up.
func ships(sinks bool) Lane {
	return Lane{Kind: River, Dir: 1, Speed: 8, Pattern: []Spawn{
		{Entity: Ferry, Gap: 120}, {Entity: Barge, Gap: 144, Sinks: sinks},
	}}
}

// Stroll's layouts, each bringing something new (DESIGN §5.7, D18): logs
// first, then barges, the ferry and lily pads, then pads that dive, then a
// ship that sinks, then both together.
var (
	strollLogs    = strollLayout("Logs", [5]Lane{logsRight, logsLeft, logsSlow, logsQuick, logsShort})
	strollBoats   = strollLayout("Boats", [5]Lane{logsRight, pads(14), ships(false), pads(16), logsShort})
	strollDivers  = strollLayout("Divers", [5]Lane{logsRight, pads(14, false, true), ships(false), pads(16, false, false, true), logsShort})
	strollSinking = strollLayout("Sinking", [5]Lane{logsRight, pads(14), ships(true), pads(16, false, true), logsShort})
	strollFlood   = strollLayout("Flood", [5]Lane{logsRight, pads(14, false, true), ships(true), pads(16, true, false), logsShort})
)

// StrollLevels is the Stroll mode's levels (DESIGN §5.7, D14). Sinking ships come
// only in its later levels, and never between rows of diving pads before
// the last (D33). Herons from level 6 (D34). Starting values, tuned in M4.
var StrollLevels = ModeLevels{
	Levels: []LevelDef{
		{Layout: strollLogs, SpeedPct: 100, GapPct: 100, Hint: "RIDE THE LOGS TO THE NOOKS ON THE FAR BANK"},
		{Layout: strollLogs, SpeedPct: 110, GapPct: 95, Hint: "FRUIT IN A NOOK IS A BONUS"},
		{Layout: strollBoats, SpeedPct: 100, GapPct: 100, Hint: "LILY PADS, A BARGE AND THE FERRY JOIN THE LOGS"},
		{Layout: strollBoats, SpeedPct: 110, GapPct: 95, Hint: "CATCH A FIREFLY FOR AN EXTRA LIFE"},
		{Layout: strollDivers, SpeedPct: 105, GapPct: 95, Hint: "LILY PADS THAT FLASH ARE ABOUT TO DIVE"},
		{Layout: strollDivers, SpeedPct: 115, GapPct: 90, HeronPct: 15, Hint: "HERONS ON THE LOGS: DON'T LAND ON ONE"},
		{Layout: strollSinking, SpeedPct: 115, GapPct: 90, HeronPct: 25, Hint: "A SINKING SHIP: RUN TO ITS STERN OR HOP OFF"},
		{Layout: strollFlood, SpeedPct: 125, GapPct: 85, HeronPct: 35},
	},
	// D20: level 38's speed, where even the tightest road gap still stays
	// open about half a second.
	CeilingPct: 225,
	// Fruit from level 2, then fireflies on logs, lily pads and ships, one
	// kind at a time (D60).
	FruitFrom:   2,
	FireflyFrom: [NumHosts]int{HostLogs: 4, HostPads: 5, HostShips: 7},
}
