package config

// Skill is a difficulty level the player picks before a game. Each one
// sets the lane speed, the time for each attempt and how forgiving the
// collision boxes are (DESIGN §5.6, D13).
type Skill struct {
	ID   string // stable key, used in saved files
	Name string // shown in menus
	Desc string // one line shown under the menu

	SpeedPct  int // lane speeds, as a percentage of the level's own
	TimerSecs int // time for each attempt
	HitMargin int // pixels trimmed from each side of a hazard's collision box
}

// Skills are the skill levels, easiest first. The values are starting
// points, tuned in M5.
var Skills = []Skill{
	{ID: "stroll", Name: "Stroll", Desc: "A gentle start: slower traffic and plenty of time.",
		SpeedPct: 80, TimerSecs: 60, HitMargin: 6},
	{ID: "dash", Name: "Dash", Desc: "The crossing as it is meant to be played.",
		SpeedPct: 100, TimerSecs: 45, HitMargin: 4},
	{ID: "rushhour", Name: "Rush Hour", Desc: "Faster lanes and less time on the clock.",
		SpeedPct: 120, TimerSecs: 40, HitMargin: 3},
	{ID: "midnight", Name: "Midnight", Desc: "Fast, tight and unforgiving.",
		SpeedPct: 140, TimerSecs: 30, HitMargin: 2},
}
