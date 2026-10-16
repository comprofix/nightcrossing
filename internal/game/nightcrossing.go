package game

import "github.com/comprofix/nightcrossing/internal/play"

// Night Crossing's place in the game: its play scene, sounds, sprite sheet,
// and its own title and high score tunes.
func init() {
	content = Content{
		NewPlay:    play.New,
		Sounds:     play.Sounds,
		Sheets:     []string{play.Sheet},
		TitleTune:  play.NightTitle,
		ScoresTune: play.HighScores,
	}
}
