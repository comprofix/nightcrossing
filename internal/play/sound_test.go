package play

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// TestSoundTables checks the game's sounds, with the shared scenes', have
// no mistakes: names, empty parts, channels, looping parts in step.
func TestSoundTables(t *testing.T) {
	if err := audio.Join(scene.Sounds, Sounds).Check(); err != nil {
		t.Fatal(err)
	}
}

// ticks is how many ticks notes last.
func ticks(notes []audio.Note) int {
	return int(math.Round(float64(audio.Length(notes)) / float64(audio.Ticks(1))))
}

// TestLoopsAreWholeBars checks every part of each looping tune is a whole
// number of 96-tick bars, so the loop comes round on the beat.
func TestLoopsAreWholeBars(t *testing.T) {
	const bar = 96
	for _, tune := range Sounds.Tunes {
		if !tune.Loop {
			continue
		}
		for i, p := range tune.Parts {
			if n := ticks(p.Notes); n%bar != 0 {
				t.Errorf("%s part %d lasts %d ticks, not whole bars of %d", tune.Name, i, n, bar)
			}
		}
	}
}

// TestEveryEventHasASound checks each world event sounds (DESIGN §3.4),
// except landing and a new attempt, which follow a sound already played,
// and the level clear and game over, which play jingles.
func TestEveryEventHasASound(t *testing.T) {
	p := &Play{hero: Toad}
	// The crossing bell rings through the warning (railEffect), not just on
	// the event, and a train leaving is quiet. A wave starting or ending is
	// quiet too: its traffic light shows it.
	quiet := map[world.Event]bool{world.Landed: true, world.Respawned: true, world.LevelClear: true, world.GameOver: true,
		world.TrainWarning: true, world.TrainGone: true,
		world.WaveStart: true, world.WaveEnd: true}
	for e := world.Hopped; e <= world.WaveEnd; e++ {
		got := p.soundsFor(e)
		switch {
		case quiet[e] && len(got) > 0:
			t.Errorf("event %d plays effects, want it quiet", e)
		case !quiet[e] && len(got) == 0:
			t.Errorf("event %d has no sound", e)
		}
	}
	for _, j := range []*audio.Tune{LevelClearJingle, GameOverJingle} {
		if j.Loop {
			t.Errorf("jingle %q loops", j.Name)
		}
	}
}

// TestMusicFollowsTheGame checks the gameplay loop starts with a level and
// the level-clear jingle replaces it when the last nook fills.
func TestMusicFollowsTheGame(t *testing.T) {
	h := scenetest.New(t, 3)
	h.Ctx.Audio = audio.NewSynth(config.Chip4)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.Modes[0], config.StartLives)
	lv := config.Level{Number: 1, TimerTicks: config.TimerTicks(1)}
	lv.Lanes[config.GoalRow].Kind = config.Bank
	p := newPlay(h.Ctx, config.StrollLevels, lv)
	n := world.NookAt(config.StartCol)
	for i := range p.World().Filled {
		p.World().Filled[i] = i != n
	}
	h.Start(p)
	h.Ticks(scene.FadeTicks + 1) // scenes start updating once faded in
	if got := h.Ctx.Audio.Tune(); got != Stroll {
		t.Fatalf("playing %v, want the gameplay loop", got)
	}
	for range config.StartRow - config.GoalRow {
		h.Run(scenetest.Hold(1, ebiten.KeyArrowUp), scenetest.Hold(config.HopTicks+1))
	}
	if got := h.Ctx.Audio.Tune(); got != LevelClearJingle {
		t.Errorf("after clearing the level playing %v, want the level-clear jingle", got)
	}
}

// TestTrainSoundsAndSignals runs Last Train's track through a warning and
// a train: the bell rings all through the warning, the rumble while it
// passes, and only the signal on the train's side flashes.
func TestTrainSoundsAndSignals(t *testing.T) {
	h := scenetest.New(t, 5)
	h.Ctx.Session = session.New(config.LastTrain, config.StartLives)
	p := newPlay(h.Ctx, config.LastTrain.Levels, config.LastTrain.Levels.Level(1))
	w := p.World()
	bells, rumbles, phase := 0, 0, world.RailQuiet
	for range 30 * config.TPS {
		w.Update(world.None)
		switch p.railEffect() {
		case Bell:
			bells++
		case Rumble:
			rumbles++
		}
		rl := w.Rail
		if rl.Phase == world.RailQuiet {
			if p.signalFrame(0) != signalOff || p.signalFrame(1) != signalOff {
				t.Fatal("a signal flashes with no train coming")
			}
		} else {
			coming, other := 0, 1
			if rl.Dir < 0 {
				coming, other = 1, 0
			}
			if p.signalFrame(coming) == signalOff || p.signalFrame(other) != signalOff {
				t.Fatalf("train from %d: signals show %d and %d", rl.Dir, p.signalFrame(0), p.signalFrame(1))
			}
		}
		if phase == world.RailPassing && rl.Phase == world.RailQuiet {
			break
		}
		phase = rl.Phase
	}
	if want := (config.TrainWarnTicks + bellTicks - 1) / bellTicks; bells != want {
		t.Errorf("%d bells in a warning, want %d", bells, want)
	}
	if rumbles == 0 {
		t.Error("no rumble while the train passed")
	}
}

// TestHornIsAChord checks the train horn sounds two notes at once, a minor
// third apart on two channels and equally long, so it is heard as a horn
// over the tune (DESIGN §6.2).
func TestHornIsAChord(t *testing.T) {
	got := (&Play{hero: Toad}).soundsFor(world.TrainComing)
	if len(got) != 2 {
		t.Fatalf("a train coming plays %d effects, want the two horn notes", len(got))
	}
	hi, lo := got[0], got[1]
	if hi.Channel == lo.Channel {
		t.Errorf("both horn notes on channel %d", hi.Channel)
	}
	if ticks(hi.Part.Notes) != ticks(lo.Part.Notes) {
		t.Errorf("horn notes last %d and %d ticks, want the same", ticks(hi.Part.Notes), ticks(lo.Part.Notes))
	}
	if r := hi.Part.Notes[0].Hz / lo.Part.Notes[0].Hz; math.Abs(r-math.Pow(2, 3.0/12)) > 0.001 {
		t.Errorf("horn notes %.1f and %.1f Hz, want a minor third apart", hi.Part.Notes[0].Hz, lo.Part.Notes[0].Hz)
	}
}
