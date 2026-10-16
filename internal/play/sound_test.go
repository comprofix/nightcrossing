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
	quiet := map[world.Event]bool{world.Landed: true, world.Respawned: true, world.LevelClear: true, world.GameOver: true}
	for e := world.Hopped; e <= world.GameOver; e++ {
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
