package play

import "github.com/comprofix/nightcrossing/internal/audio"

// The game's sounds. Channels follow the shared tune layout (see
// scene.TitleTune): 0 lead, 1 harmony, 2 bass, 3 noise. These are
// placeholders, finished in M7.
var (
	// ToadHop is the toad leaving a cell: a short rising blip.
	ToadHop = &audio.Effect{Name: "toad hop", Channel: 1, Priority: 1, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(4), Volume: 0.5},
		Notes: audio.Notes("E5:2 B5:3"),
	}}
	// Crash is a vehicle hitting the hero.
	Crash = &audio.Effect{Name: "crash", Channel: 3, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(20), Release: audio.Ticks(4)},
		Notes: []audio.Note{{Hz: 700, Dur: audio.Ticks(20)}},
	}}
	// GameOver is the end of a game.
	GameOver = &audio.Effect{Name: "game over", Channel: 0, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(20), Sustain: 0.5, Release: audio.Ticks(12), Volume: 0.8},
		Notes: audio.Notes("E5:10 C5:10 A4:10 -:4 E4:36"),
	}}
)

// Sounds are the game's tunes and effects, listed for the sound test.
var Sounds = audio.Bank{Effects: []*audio.Effect{ToadHop, Crash, GameOver}}
