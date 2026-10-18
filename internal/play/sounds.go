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
	// Bell is the crossing bell, rung again and again while the lights
	// flash (DESIGN §3.4, §6.2): a bright two-note ding.
	Bell = &audio.Effect{Name: "crossing bell", Channel: 1, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(10), Sustain: 0.2, Release: audio.Ticks(4), Volume: 0.55},
		Notes: audio.Notes("E6:3 C6:9"),
	}}
	// Horn is the train coming onto the field: one long blast of two notes
	// held together a minor third apart, as a train horn sounds, so it
	// stands out from the tune. HornLow is its lower note, on the bass
	// channel; the two always play together.
	Horn = &audio.Effect{Name: "train horn", Channel: 0, Priority: 4, Part: audio.Part{
		Voice: hornVoice(audio.Pulse25), Notes: audio.Notes("Eb4:48"),
	}}
	HornLow = &audio.Effect{Name: "train horn low", Channel: 2, Priority: 4, Part: audio.Part{
		Voice: hornVoice(audio.Pulse50), Notes: audio.Notes("C4:48"),
	}}
	// Rumble is the train on the track, repeated while it passes: a low,
	// rough roll of noise.
	Rumble = &audio.Effect{Name: "train rumble", Channel: 3, Priority: 2, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(10), Sustain: 0.5, Release: audio.Ticks(2), Volume: 0.6},
		Notes: []audio.Note{{Hz: 180, Dur: audio.Ticks(4)}, {Hz: 120, Dur: audio.Ticks(8)}},
	}}
	// TrainCrash is a train hitting the hero (D37): a bigger, longer crash
	// than a car's.
	TrainCrash = &audio.Effect{Name: "train crash", Channel: 3, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(40), Release: audio.Ticks(8)},
		Notes: []audio.Note{{Hz: 3000, Dur: audio.Ticks(3)}, {Hz: 1400, Dur: audio.Ticks(6)}, {Hz: 600, Dur: audio.Ticks(12)}, {Hz: 250, Dur: audio.Ticks(24)}},
	}}
	// ComboUp is the hop combo's multiplier rising (D38): a quick bright
	// run up, on the hop's channel so it follows the hop.
	ComboUp = &audio.Effect{Name: "combo up", Channel: 1, Priority: 2, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(6), Sustain: 0.3, Release: audio.Ticks(4), Volume: 0.5},
		Notes: audio.Notes("G5:3 C6:3 E6:3 G6:6"),
	}}
	// HornChorus is a rush hour wave being announced (D43): impatient car
	// horns, two at once in different rhythms. HornChorusLow is the second
	// horn, on the bass channel; the two always play together.
	HornChorus = &audio.Effect{Name: "horn chorus", Channel: 0, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Attack: audio.Ticks(1), Sustain: 0.9, Release: audio.Ticks(3), Volume: 0.55},
		Notes: audio.Notes("A4:5 -:3 A4:5 -:10 A4:14"),
	}}
	HornChorusLow = &audio.Effect{Name: "horn chorus low", Channel: 2, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Attack: audio.Ticks(1), Sustain: 0.9, Release: audio.Ticks(3), Volume: 0.5},
		Notes: audio.Notes("-:6 E4:10 -:4 F4:4 -:3 F4:12"),
	}}
	// FogHorn is fog being announced (D29): a long low horn, twice, from
	// somewhere out in the murk.
	FogHorn = &audio.Effect{Name: "fog horn", Channel: 2, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Triangle, Attack: audio.Ticks(6), Sustain: 1, Release: audio.Ticks(10), Volume: 0.9},
		Notes: audio.Notes("D3:36 -:10 D3:44"),
	}}
	// Thunder is rain being announced (D31): a crack, then a long roll.
	Thunder = &audio.Effect{Name: "thunder", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(60), Sustain: 0.2, Release: audio.Ticks(20), Volume: 0.9},
		Notes: []audio.Note{{Hz: 1800, Dur: audio.Ticks(4)}, {Hz: 300, Dur: audio.Ticks(30)}, {Hz: 160, Dur: audio.Ticks(50)}},
	}}
	// Skid is the hero sliding on the wet road (D31): a short hiss that
	// drops away.
	Skid = &audio.Effect{Name: "skid", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(10), Release: audio.Ticks(2), Volume: 0.6},
		Notes: []audio.Note{{Hz: 5000, Dur: audio.Ticks(3)}, {Hz: 3000, Dur: audio.Ticks(5)}},
	}}
	// Howl is a gust spell being announced (D47): the wind rising, holding
	// and dying away, a long breath of noise.
	Howl = &audio.Effect{Name: "gust howl", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Attack: audio.Ticks(24), Sustain: 0.8, Release: audio.Ticks(30), Volume: 0.6},
		Notes: []audio.Note{{Hz: 500, Dur: audio.Ticks(20)}, {Hz: 800, Dur: audio.Ticks(30)}, {Hz: 650, Dur: audio.Ticks(30)}, {Hz: 420, Dur: audio.Ticks(30)}},
	}}
	// Whoosh is a gust pushing the hero a cell (D47): a quick rush of air.
	Whoosh = &audio.Effect{Name: "whoosh", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Attack: audio.Ticks(3), Decay: audio.Ticks(12), Release: audio.Ticks(4), Volume: 0.6},
		Notes: []audio.Note{{Hz: 1200, Dur: audio.Ticks(3)}, {Hz: 2200, Dur: audio.Ticks(5)}, {Hz: 1400, Dur: audio.Ticks(8)}},
	}}
	// Rising is a flood being announced (D48): water rushing in, a low swell
	// that grows and runs on.
	Rising = &audio.Effect{Name: "rising water", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Attack: audio.Ticks(30), Sustain: 0.9, Release: audio.Ticks(30), Volume: 0.7},
		Notes: []audio.Note{{Hz: 260, Dur: audio.Ticks(30)}, {Hz: 380, Dur: audio.Ticks(30)}, {Hz: 320, Dur: audio.Ticks(40)}},
	}}
	// DistantThunder is the storm sky's thunder, a beat after a flash (D49):
	// a low roll with no crack, unlike rain's warning, and quieter than any
	// cue, so it never covers one.
	DistantThunder = &audio.Effect{Name: "distant thunder", Channel: 3, Priority: 1, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Attack: audio.Ticks(8), Decay: audio.Ticks(70), Sustain: 0.2, Release: audio.Ticks(20), Volume: 0.5},
		Notes: []audio.Note{{Hz: 140, Dur: audio.Ticks(40)}, {Hz: 90, Dur: audio.Ticks(60)}},
	}}
	// Squawk is a heron catching the hero: a harsh falling cry.
	Squawk = &audio.Effect{Name: "squawk", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(16), Sustain: 0.3, Release: audio.Ticks(6), Volume: 0.7},
		Notes: audio.Notes("D6:3 A5:4 E5:6 B4:10"),
	}}
)

// The predators' sounds (D67, D68). Each warning is quiet and short, so it
// never covers a cue, but plain enough to hear coming.
var (
	// Clack is a heron cocking its head, the warning before it strikes: two
	// dry knocks of its beak.
	Clack = &audio.Effect{Name: "beak clack", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(3), Release: audio.Ticks(1), Volume: 0.5},
		Notes: []audio.Note{{Hz: 1800, Dur: audio.Ticks(3)}, audio.Rest(3), {Hz: 1600, Dur: audio.Ticks(4)}},
	}}
	// Rustle is a fox's eyes glinting in a nook: something stirring in the
	// grass.
	Rustle = &audio.Effect{Name: "rustle", Channel: 3, Priority: 2, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Attack: audio.Ticks(6), Sustain: 0.5, Release: audio.Ticks(10), Volume: 0.35},
		Notes: []audio.Note{{Hz: 900, Dur: audio.Ticks(8)}, {Hz: 700, Dur: audio.Ticks(10)}},
	}}
	// Yip is the fox coming into its nook: a short high bark.
	Yip = &audio.Effect{Name: "yip", Channel: 2, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(6), Sustain: 0.2, Release: audio.Ticks(3), Volume: 0.5},
		Notes: audio.Notes("A5:3 E6:5"),
	}}
	// Snarl is the fox snatching the hero: a low growl, falling.
	Snarl = &audio.Effect{Name: "snarl", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(16), Sustain: 0.4, Release: audio.Ticks(8), Volume: 0.7},
		Notes: audio.Notes("E4:4 G4:3 D4:6 A3:12"),
	}}
)

// The toadlets' sounds (D56).
var (
	// ToadletPeep is a toadlet left behind at the verge, or getting home: a
	// tiny high chirp.
	ToadletPeep = &audio.Effect{Name: "toadlet peep", Channel: 1, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(3), Volume: 0.45},
		Notes: audio.Notes("G6:2 C7:3"),
	}}
	// ToadletCry is a toadlet dying: a high falling wail, before the death's
	// own sound.
	ToadletCry = &audio.Effect{Name: "toadlet cry", Channel: 0, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(10), Sustain: 0.3, Release: audio.Ticks(4), Volume: 0.6},
		Notes: audio.Notes("C7:3 B6:3 G6:4 D6:6"),
	}}
)

// The pickups' sounds (D58, D59). Each plays on the harmony channel, so it
// sounds with the 1UP or the nook's home run on the lead.
var (
	// FireflyTwinkle is a firefly caught: a soft high twinkle.
	FireflyTwinkle = &audio.Effect{Name: "firefly", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(5), Sustain: 0.4, Release: audio.Ticks(8), Volume: 0.55},
		Notes: audio.Notes("E6:2 B6:2 E7:3 -:2 B6:2 E7:8"),
	}}
	// DragonflyBuzz is a dragonfly caught: a quick buzzing trill, then a
	// chirp.
	DragonflyBuzz = &audio.Effect{Name: "dragonfly", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse12, Decay: audio.Ticks(4), Sustain: 0.5, Release: audio.Ticks(4), Volume: 0.5},
		Notes: audio.Notes("A5:1 C6:1 A5:1 C6:1 A5:1 C6:1 E6:2 A6:6"),
	}}
	// FruitPick is fruit taken with its nook: a bright sparkle just after
	// the home run starts.
	FruitPick = &audio.Effect{Name: "fruit", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(4), Sustain: 0.4, Release: audio.Ticks(6), Volume: 0.5},
		Notes: audio.Notes("-:4 G6:3 C7:3 E7:8"),
	}}
)

// The bonus level's sounds (D61): softer than a death's, since a slip
// costs only time, and a parked car only a moment.
var (
	// Slip is a slip on a bonus level, into the water or a car: a soft
	// falling plop.
	Slip = &audio.Effect{Name: "slip", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(8), Sustain: 0.3, Release: audio.Ticks(6), Volume: 0.6},
		Notes: audio.Notes("C5:3 G4:3 C4:10"),
	}}
	// Blocked is a hop into a parked car, refused: a small dull knock.
	Blocked = &audio.Effect{Name: "blocked", Channel: 3, Priority: 2, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(4), Release: audio.Ticks(1), Volume: 0.45},
		Notes: []audio.Note{{Hz: 220, Dur: audio.Ticks(4)}},
	}}
)

// Sounds are the game's tunes and effects, listed for the sound test.
var Sounds = audio.Bank{
	Tunes:   []*audio.Tune{NightTitle, Stroll, BonusNight, LevelClearJingle, GameOverJingle, HighScores},
	Effects: []*audio.Effect{ToadHop, Crash, Bump, Splash, Bubbles, Squawk, Home, OneUp, TimeLow, TimeUp, Bell, Horn, HornLow, Rumble, TrainCrash, ComboUp, HornChorus, HornChorusLow, FogHorn, Thunder, Skid, Howl, Whoosh, Rising, DistantThunder, ToadletPeep, ToadletCry, FireflyTwinkle, DragonflyBuzz, FruitPick, Slip, Blocked, Clack, Rustle, Yip, Snarl},
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
	case world.Snatched:
		return []*audio.Effect{Snarl}
	case world.HeronCocks:
		return []*audio.Effect{Clack}
	case world.EyesGlint:
		return []*audio.Effect{Rustle}
	case world.FoxIn:
		return []*audio.Effect{Yip}
	case world.Struck:
		return []*audio.Effect{TrainCrash}
	case world.TrainComing:
		return []*audio.Effect{Horn, HornLow}
	case world.ComboUp:
		return []*audio.Effect{ComboUp}
	case world.WaveWarning:
		return []*audio.Effect{HornChorus, HornChorusLow}
	case world.FogWarning:
		return []*audio.Effect{FogHorn}
	case world.RainWarning:
		return []*audio.Effect{Thunder}
	case world.Slid:
		return []*audio.Effect{Skid}
	case world.WetStormWarning:
		return []*audio.Effect{Thunder}
	case world.DryStormWarning:
		return []*audio.Effect{Howl}
	case world.Pushed:
		return []*audio.Effect{Whoosh}
	case world.FloodStart:
		return []*audio.Effect{Rising}
	case world.TimeUp:
		return []*audio.Effect{TimeUp}
	case world.LowTime:
		return []*audio.Effect{TimeLow}
	case world.Goal:
		return []*audio.Effect{Home}
	case world.Refused, world.Joined:
		return []*audio.Effect{p.hero.Peep}
	case world.ExtraLife:
		return []*audio.Effect{OneUp}
	case world.FireflyCaught:
		if p.world != nil && (p.world.Flyer.Day || p.world.Bonus() && p.world.Level.Bonus.Dawn) { // a bonus level's dragonflies by dawn
			return []*audio.Effect{p.hero.DayCatch}
		}
		return []*audio.Effect{p.hero.Catch}
	case world.FruitTaken:
		return []*audio.Effect{FruitPick}
	case world.Slipped:
		return []*audio.Effect{Slip}
	case world.Blocked:
		return []*audio.Effect{Blocked}
	}
	return nil
}

// hornVoice is one note of the train horn: a quick swell, held full, and a
// short fade.
func hornVoice(w audio.Wave) audio.Voice {
	return audio.Voice{Wave: w, Attack: audio.Ticks(3), Sustain: 1, Release: audio.Ticks(10), Volume: 0.75}
}
