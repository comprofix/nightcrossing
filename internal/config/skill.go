package config

// Skill is what the player picks before a game. Until the modes arrive in
// M7 it stands in for a mode: every game uses the same base numbers (D13),
// so there is one skill, Stroll, the only mode built so far (DESIGN §5.6,
// D26).
type Skill struct {
	ID   string // stable key, used in saved files
	Name string // shown in menus
	Desc string // one line shown under the menu
}

// Skills are the skills to pick from.
var Skills = []Skill{
	{ID: "stroll", Name: "Stroll", Desc: "Road, verge and river: get five toads home."},
}
