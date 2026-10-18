package config

// The game's ending, after the school gates' tally (D62, DESIGN §6.7): a
// heading and a line of thanks over the toad and its toadlets home, then
// the credits rolling up, then the MASTERED screen. EVERY ONE HOME only
// when every straggler was brought home.
const (
	EndingHeadingAll = "EVERY ONE HOME"
	EndingHeading    = "HOME AT LAST"
	EndingThanks     = "THANK YOU FOR PLAYING"
)

// Credits roll up the ending, a line each; "" is a gap. The palette is
// credited as its author asked (DESIGN §3).
var Credits = []string{
	"NIGHT CROSSING",
	"",
	"A GAME BY",
	"MATTHEW MCKINNON",
	"",
	"PALETTE",
	"RESURRECT 64 BY KERRIE LAKE",
	"",
	"MADE WITH",
	"GO AND EBITENGINE",
	"",
	"",
	"THE END",
}
