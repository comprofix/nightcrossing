package play

import (
	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The game's sound effects (DESIGN §3.4). Channels follow the tune layout
// (see music.go): 0 lead, 1 harmony, 2 bass, 3 noise. An effect takes over
// its channel while it plays and the music comes back in step after.
var (
	// ToadHop is the toad leaving a cell: a short rising blip.
	ToadHop = &audio.Effect{Name: "toad hop", Channel: 1, Priority: 1, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(4), Volume: 0.5},
		Notes: audio.Notes("E5:2 B5:3"),
	}}
	// Crash is a vehicle hitting the hero: a hard crack that drops to a
	// rumble.
	Crash = &audio.Effect{Name: "crash", Channel: 3, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(24), Release: audio.Ticks(4)},
		Notes: []audio.Note{{Hz: 2000, Dur: audio.Ticks(3)}, {Hz: 900, Dur: audio.Ticks(6)}, {Hz: 400, Dur: audio.Ticks(16)}},
	}}
	// Bump is the hero landing on the bank or a filled nook: a dull thud,
	// lower and shorter than a crash.
	Bump = &audio.Effect{Name: "bump", Channel: 3, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(10), Release: audio.Ticks(2), Volume: 0.8},
		Notes: []audio.Note{{Hz: 260, Dur: audio.Ticks(4)}, {Hz: 150, Dur: audio.Ticks(10)}},
	}}
	// TimeUp is the attempt's time running out: a falling buzz.
	TimeUp = &audio.Effect{Name: "time up", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(24), Sustain: 0.4, Release: audio.Ticks(6), Volume: 0.7},
		Notes: audio.Notes("A5:6 F5:6 D5:6 A4:24"),
	}}
	// Splash is the hero falling in the water: a soft hiss that sweeps from
	// bright to low, unlike the crash's single hard hit. Bubbles plays with it.
	Splash = &audio.Effect{Name: "splash", Channel: 3, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Attack: audio.Ticks(2), Decay: audio.Ticks(30), Sustain: 0.5,
			Release: audio.Ticks(10), Volume: 0.7},
		Notes: []audio.Note{
			{Hz: 7000, Dur: audio.Ticks(3)}, {Hz: 5000, Dur: audio.Ticks(3)}, {Hz: 3500, Dur: audio.Ticks(4)},
			{Hz: 2400, Dur: audio.Ticks(5)}, {Hz: 1600, Dur: audio.Ticks(6)}, {Hz: 1100, Dur: audio.Ticks(8)},
			{Hz: 800, Dur: audio.Ticks(10)},
		},
	}}
	// Bubbles is air coming up after a splash: quick rising blips.
	Bubbles = &audio.Effect{Name: "bubbles", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(3), Volume: 0.6},
		Notes: audio.Notes("-:8 C5:2 G5:2 -:4 E5:2 C6:2 -:6 A4:2 E5:2 -:8 D5:2 A5:2"),
	}}
	// Home is a toad reaching a nook: a bright rising run.
	Home = &audio.Effect{Name: "home", Channel: 0, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(6), Sustain: 0.4, Release: audio.Ticks(6), Volume: 0.6},
		Notes: audio.Notes("C5:4 E5:4 G5:4 C6:10"),
	}}
	// OneUp is an extra life.
	OneUp = &audio.Effect{Name: "extra life", Channel: 0, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(4), Sustain: 0.5, Release: audio.Ticks(6), Volume: 0.6},
		Notes: audio.Notes("G5:4 C6:4 G5:4 C6:4 E6:12"),
	}}
	// TimeLow warns that the attempt's time is running low (DESIGN §5.4).
	TimeLow = &audio.Effect{Name: "time low", Channel: 1, Priority: 2, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(6), Volume: 0.5},
		Notes: audio.Notes("A5:6 -:6 A5:6 -:6 A5:6"),
	}}
	// Squawk is a heron catching the hero: a harsh falling cry.
	Squawk = &audio.Effect{Name: "squawk", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(16), Sustain: 0.3, Release: audio.Ticks(6), Volume: 0.7},
		Notes: audio.Notes("D6:3 A5:4 E5:6 B4:10"),
	}}
)

// Sounds are the game's tunes and effects, listed for the sound test.
var Sounds = audio.Bank{
	Tunes:   []*audio.Tune{NightTitle, Stroll, LevelClearJingle, GameOverJingle, HighScores},
	Effects: []*audio.Effect{ToadHop, Crash, Bump, Splash, Bubbles, Squawk, Home, OneUp, TimeLow, TimeUp},
}

// soundsFor is the effects an event plays (DESIGN §3.4). The level clear
// and game over play jingles instead (see music.go), and landing and a new
// attempt are quiet: the hop and the death already sounded.
func (p *Play) soundsFor(e world.Event) []*audio.Effect {
	switch e {
	case world.Hopped:
		return []*audio.Effect{p.hero.Hop}
	case world.Squashed:
		return []*audio.Effect{Crash}
	case world.Bumped:
		return []*audio.Effect{Bump}
	case world.Splashed, world.Swept:
		return []*audio.Effect{Splash, Bubbles}
	case world.Caught:
		return []*audio.Effect{Squawk}
	case world.TimeUp:
		return []*audio.Effect{TimeUp}
	case world.LowTime:
		return []*audio.Effect{TimeLow}
	case world.Goal:
		return []*audio.Effect{Home}
	case world.ExtraLife:
		return []*audio.Effect{OneUp}
	}
	return nil
}
