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
)

// Sounds are the game's tunes and effects, listed for the sound test.
var Sounds = audio.Bank{Effects: []*audio.Effect{ToadHop}}
