package play

import "github.com/comprofix/nightcrossing/internal/audio"

// Night Crossing's music (DESIGN §3.4: a title tune, a gameplay loop, a
// level-clear jingle and a high-score tune, all new). Part i plays on
// channel i: 0 lead, 1 harmony, 2 bass, 3 drums. Bars are 96 ticks, 4/4
// with an eighth note of 12 ticks.

// musicLevel scales every music voice, so the music sits under the sound
// effects at the default volumes: at 80% music it sounds as the tunes
// did at 20%, the balance chosen in play.
const musicLevel = 0.25

// Voices for the music.
var (
	tuneLead  = audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(10), Sustain: 0.55, Release: audio.Ticks(3), Volume: 0.7 * musicLevel}
	tunePad   = audio.Voice{Wave: audio.Pulse12, Attack: audio.Ticks(4), Release: audio.Ticks(8), Volume: 0.3 * musicLevel}
	tuneBass  = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(12), Sustain: 0.7, Release: audio.Ticks(2), Volume: 0.9 * musicLevel}
	tuneDrums = audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(5), Release: audio.Ticks(1), Volume: 0.35 * musicLevel}
	fanfare   = audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(8), Sustain: 0.6, Release: audio.Ticks(6), Volume: 0.7 * musicLevel}
)

// Drum sounds, as noise rates: low, middle and high. 0 is a rest.
const (
	kick  = 300
	snare = 3000
	hat   = 12000
)

// beats makes drum notes from a pattern of eighth notes, repeated.
func beats(times int, pattern ...float64) []audio.Note {
	var out []audio.Note
	for range times {
		for _, hz := range pattern {
			out = append(out, audio.Note{Hz: hz, Dur: audio.Ticks(12)})
		}
	}
	return out
}

var (
	// NightTitle loops on the title and menus: a moody walk in A minor,
	// Am F C G | Am F Dm E.
	NightTitle = &audio.Tune{Name: "night title", Loop: true, Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes(
			"E5:24 A5:24 C6:36 B5:12",
			"A5:24 F5:24 C5:48",
			"E5:12 G5:12 C6:24 B5:12 C6:12 D6:24",
			"B5:48 G5:24 -:24",
			"E5:24 A5:24 C6:24 E6:24",
			"D6:24 C6:12 A5:12 F5:48",
			"F5:12 A5:12 D6:24 C6:12 A5:12 F5:24",
			"E5:24 G#5:24 B5:48",
		)},
		{Voice: tunePad, Notes: audio.Notes(
			"C5:48 E5:48", "A4:48 C5:48", "E4:48 G4:48", "D5:48 B4:48",
			"C5:48 E5:48", "A4:48 C5:48", "D5:48 F5:48", "G#4:48 B4:48",
		)},
		{Voice: tuneBass, Notes: audio.Notes(
			"A2:24 E3:24 A2:24 E3:24", "F2:24 C3:24 F2:24 C3:24",
			"C3:24 G2:24 C3:24 G2:24", "G2:24 D3:24 G2:24 D3:24",
			"A2:24 E3:24 A2:24 E3:24", "F2:24 C3:24 F2:24 C3:24",
			"D3:24 A2:24 D3:24 A2:24", "E2:24 B2:24 E2:24 G#2:24",
		)},
		{Voice: tuneDrums, Notes: beats(8, kick, 0, hat, 0, snare, 0, hat, hat)},
	}}

	// Stroll loops while a level is played: a brisk tune in D minor,
	// Dm Bb F C | Dm Bb Gm A.
	Stroll = &audio.Tune{Name: "stroll", Loop: true, Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes(
			"D5:12 F5:12 A5:24 G5:12 F5:12 E5:12 D5:12",
			"F5:12 D5:12 Bb4:24 C5:12 D5:12 F5:24",
			"A5:12 C6:12 A5:12 F5:12 G5:24 A5:24",
			"G5:12 E5:12 C5:24 E5:12 G5:12 -:24",
			"D5:12 F5:12 A5:24 D6:24 C6:12 A5:12",
			"Bb5:24 A5:12 G5:12 F5:24 D5:24",
			"G5:12 Bb5:12 D6:24 C6:12 Bb5:12 A5:12 G5:12",
			"A5:24 C#6:12 E6:12 A5:24 -:24",
		)},
		{Voice: tunePad, Notes: audio.Notes(
			"F4:48 A4:48", "D4:48 F4:48", "A4:48 C5:48", "G4:48 E4:48",
			"F4:48 A4:48", "D4:48 F4:48", "Bb4:48 D5:48", "C#5:48 E5:48",
		)},
		{Voice: tuneBass, Notes: audio.Notes(
			"D3:24 A2:24 D3:24 F3:24", "Bb2:24 F2:24 Bb2:24 D3:24",
			"F2:24 C3:24 F2:24 A2:24", "C3:24 G2:24 C3:24 E3:24",
			"D3:24 A2:24 D3:24 F3:24", "Bb2:24 F2:24 Bb2:24 D3:24",
			"G2:24 D3:24 G2:24 Bb2:24", "A2:24 E3:24 A2:24 C#3:24",
		)},
		{Voice: tuneDrums, Notes: beats(8, kick, hat, snare, hat, kick, kick, snare, hat)},
	}}

	// BonusNight loops on every bonus level (D61): a light, twinkling tune
	// in F, F Am Bb C | F Dm Gm C.
	BonusNight = &audio.Tune{Name: "bonus", Loop: true, Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes(
			"F5:12 A5:12 C6:24 A5:12 C6:12 F6:24",
			"E6:24 C6:12 A5:12 E5:24 A5:24",
			"D6:12 F6:12 D6:12 Bb5:12 F5:24 Bb5:24",
			"C6:24 G5:12 E5:12 G5:48",
			"A5:12 C6:12 F6:24 E6:12 C6:12 A5:24",
			"F6:24 D6:12 A5:12 F5:24 A5:24",
			"Bb5:12 D6:12 G6:24 F6:12 D6:12 Bb5:24",
			"G5:24 Bb5:12 E6:12 C6:24 -:24",
		)},
		{Voice: tunePad, Notes: audio.Notes(
			"A4:48 C5:48", "C5:48 E5:48", "D5:48 F5:48", "E4:48 G4:48",
			"A4:48 C5:48", "D5:48 F5:48", "Bb4:48 D5:48", "E5:48 G4:48",
		)},
		{Voice: tuneBass, Notes: audio.Notes(
			"F2:24 C3:24 F2:24 C3:24", "A2:24 E3:24 A2:24 E3:24",
			"Bb2:24 F3:24 Bb2:24 F3:24", "C3:24 G2:24 C3:24 G2:24",
			"F2:24 C3:24 F2:24 A2:24", "D3:24 A2:24 D3:24 F3:24",
			"G2:24 D3:24 G2:24 Bb2:24", "C3:24 G2:24 E3:24 C3:24",
		)},
		{Voice: tuneDrums, Notes: beats(8, kick, hat, 0, hat, snare, hat, 0, hat)},
	}}

	// LevelClearJingle plays once as the last nook fills: a bright run up
	// to a held D major chord.
	LevelClearJingle = &audio.Tune{Name: "level clear", Parts: []audio.Part{
		{Voice: fanfare, Notes: audio.Notes("D5:6 F#5:6 A5:6 D6:12 C#6:6 D6:6 F#6:36")},
		{Voice: tunePad, Notes: audio.Notes("A4:18 -:6 A4:18 D5:36")},
		{Voice: tuneBass, Notes: audio.Notes("D3:18 -:6 A2:18 D3:36")},
		{Voice: tuneDrums, Notes: []audio.Note{
			{Hz: kick, Dur: audio.Ticks(18)}, {Hz: snare, Dur: audio.Ticks(6)},
			{Hz: snare, Dur: audio.Ticks(18)}, {Hz: kick, Dur: audio.Ticks(36)},
		}},
	}}

	// GameOverJingle plays once when the last life is gone: a slow fall to
	// a low, held note.
	GameOverJingle = &audio.Tune{Name: "game over", Parts: []audio.Part{
		{Voice: fanfare, Notes: audio.Notes("A5:12 E5:12 C5:12 A4:12 -:6 G#4:12 A4:48")},
		{Voice: tunePad, Notes: audio.Notes("C5:48 -:6 B4:12 C5:48")},
		{Voice: tuneBass, Notes: audio.Notes("A2:48 -:6 E2:12 A2:48")},
	}}

	// HighScores loops on the high score screens: a cheerful climb in C,
	// C F G C.
	HighScores = &audio.Tune{Name: "high scores", Loop: true, Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes(
			"C5:12 E5:12 G5:12 C6:12 B5:12 G5:12 E5:24",
			"F5:12 A5:12 C6:12 F6:12 E6:12 C6:12 A5:24",
			"G5:12 B5:12 D6:12 G6:12 F6:12 D6:12 B5:24",
			"C6:24 G5:24 C6:48",
		)},
		{Voice: tunePad, Notes: audio.Notes("E4:48 G4:48", "A4:48 C5:48", "B4:48 D5:48", "E5:48 G4:48")},
		{Voice: tuneBass, Notes: audio.Notes(
			"C3:24 G2:24 C3:24 G2:24", "F2:24 C3:24 F2:24 C3:24",
			"G2:24 D3:24 G2:24 D3:24", "C3:24 G2:24 C3:48",
		)},
		{Voice: tuneDrums, Notes: beats(4, kick, hat, snare, hat, kick, hat, snare, snare)},
	}}
)
