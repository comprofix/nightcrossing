package game

import "github.com/comprofix/nightcrossing/internal/play"

// Night Crossing's place in the game: its play scene, sounds and sprite
// sheet.
func init() {
	content = Content{
		NewPlay: play.New,
		Sounds:  play.Sounds,
		Sheets:  []string{play.Sheet},
	}
}
