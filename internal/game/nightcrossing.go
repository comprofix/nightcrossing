package game

import "github.com/comprofix/nightcrossing/internal/play"

// Night Crossing's place in the game: its play scene and title demo, its
// sounds and sprite sheet, and its own title and high score tunes.
func init() {
	content = Content{
		NewPlay:    play.New,
		NewDemo:    play.NewDemo,
		Sounds:     play.Sounds,
		Sheets:     []string{play.Sheet},
		TitleTune:  play.NightTitle,
		ScoresTune: play.HighScores,
		NativePlay: true, // play and the demo draw in either look (DESIGN §3.6)
	}
}
