package scene

import (
	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// The shared scenes' sounds: only effects; the music is the game's own
// (Context.TitleTune and ScoresTune). Menu sounds take channel 1, a
// tune's harmony in Chip4, so the melody carries on under them.
var (
	// MenuMove is the cursor moving or a value changing.
	MenuMove = &audio.Effect{Name: "menu move", Channel: 1, Priority: 1, Part: audio.Part{
		Voice: blip,
		Notes: []audio.Note{audio.N("E6", 3)},
	}}
	// MenuConfirm is a menu item chosen.
	MenuConfirm = &audio.Effect{Name: "menu confirm", Channel: 1, Priority: 2, Part: audio.Part{
		Voice: blip,
		Notes: []audio.Note{audio.N("C6", 3), audio.N("G6", 6)},
	}}

	// TallyTick is a point counting up on the tally.
	TallyTick = &audio.Effect{Name: "tally tick", Channel: 1, Priority: 1, Part: audio.Part{
		Voice: blip,
		Notes: []audio.Note{audio.N("A6", 2)},
	}}
)

// Sounds are the shared scenes' sounds, for the sound test.
var Sounds = audio.Bank{
	Effects: []*audio.Effect{MenuMove, MenuConfirm, TallyTick},
}

// blip is the menu sounds' voice.
var blip = audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(6), Sustain: 0.3, Release: audio.Ticks(2), Volume: 0.8}

// menuSounds gives m the menu sounds.
func menuSounds(ctx *Context, m *ui.Menu) {
	m.OnMove = func() { ctx.Audio.PlayEffect(MenuMove) }
	m.OnConfirm = func() { ctx.Audio.PlayEffect(MenuConfirm) }
}

// playTitleTune starts the game's title tune, if it has one, unless it is
// already playing.
func playTitleTune(ctx *Context) {
	if t := ctx.TitleTune; t != nil && ctx.Audio.Tune() != t {
		ctx.Audio.PlayTune(t)
	}
}

// playScoresTune starts the game's high score tune, if it has one, unless
// it is already playing.
func playScoresTune(ctx *Context) {
	if t := ctx.ScoresTune; t != nil && ctx.Audio.Tune() != t {
		ctx.Audio.PlayTune(t)
	}
}
