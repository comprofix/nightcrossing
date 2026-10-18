package play

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// afterTally shows q's tally, counts it through, and is the level it goes
// on to.
func afterTally(t *testing.T, h *scenetest.Harness, q *Play) *Play {
	t.Helper()
	h.Start(q.tally(h.Ctx))
	h.Ticks(scene.FadeTicks)
	h.Press(ebiten.KeyEnter) // finish the count
	h.Press(ebiten.KeyEnter) // go on
	return scenetest.Current[*Play](h)
}

// bonusHarness is a game in mode at the end of its level 8, its tally
// showing.
func bonusHarness(t *testing.T, mode config.Mode) *scenetest.Harness {
	h := scenetest.New(t, 9)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(mode, config.StartLives)
	h.Start(newLevel(h.Ctx, mode.Levels, len(mode.Levels.Levels)).tally(h.Ctx))
	h.Ticks(scene.FadeTicks)
	h.Press(ebiten.KeyEnter)
	h.Press(ebiten.KeyEnter)
	return h
}

// endBonus runs the bonus level showing in h out of time, and through its
// tally.
func endBonus(t *testing.T, h *scenetest.Harness) {
	t.Helper()
	p := scenetest.Current[*Play](h)
	h.Ticks(p.World().TicksLeft + clearTicks + 1)
	if p.ending != world.BonusOver {
		t.Fatalf("the bonus level ended with %v, want its time up", p.ending)
	}
	h.Ticks(scene.FadeTicks)
	scenetest.Current[*scene.Tally](h)
	h.Ticks(scene.FadeTicks) // its fade in
	h.Press(ebiten.KeyEnter)
	h.Press(ebiten.KeyEnter)
}

// TestLevel8LeadsToTheBonus: Stroll's level 8 goes on to firefly night,
// then its tally, then the MASTERED screen, which masters Stroll and so
// opens Last Train (D61, D62, D64).
func TestLevel8LeadsToTheBonus(t *testing.T) {
	h := bonusHarness(t, config.Stroll)
	p := scenetest.Current[*Play](h)
	if b := p.World().Level.Bonus; b != config.FireflyNight {
		t.Fatalf("after level 8: bonus %v, want firefly night", b)
	}
	endBonus(t, h)
	scenetest.Current[*Mastered](h)
	if !h.Ctx.Scores.IsMastered(config.Stroll.ID) || !h.Ctx.TakeScoresChanged() || !h.Ctx.Session.Mastered {
		t.Error("the MASTERED screen didn't master Stroll and save it")
	}
	if !h.Ctx.Scores.Open(config.Modes, 1) {
		t.Error("mastering Stroll didn't open Last Train")
	}
}

// TestNoBonusYetGoesToMastered: a mode with no bonus level built yet goes
// from level 8's tally straight to the MASTERED screen (D61).
func TestNoBonusYetGoesToMastered(t *testing.T) {
	h := bonusHarness(t, config.RushHour)
	scenetest.Current[*Mastered](h)
	if !h.Ctx.Scores.IsMastered(config.RushHour.ID) {
		t.Error("Rush Hour isn't mastered")
	}
}

// TestFinish: Finish counts up 1,000 for each life left, then the high
// score entry with the run marked MASTERED, then the next mode's intro
// (D62, D63).
func TestFinish(t *testing.T) {
	h := bonusHarness(t, config.RushHour)
	before := h.Ctx.Session.Score
	h.Press(ebiten.KeyEnter) // Finish
	scenetest.Current[*scene.Tally](h)
	if want := before + config.StartLives*config.PointsFinishLife; h.Ctx.Session.Score != want {
		t.Errorf("score %d after Finish, want %d", h.Ctx.Session.Score, want)
	}
	h.Press(ebiten.KeyEnter)
	h.Press(ebiten.KeyEnter)
	scenetest.Current[*scene.HighScoreEntry](h)
	for range config.InitialsLen {
		h.Press(ebiten.KeyEnter)
	}
	scenetest.Current[*scene.HighScoreTable](h)
	if es := h.Ctx.Scores.Table(config.RushHour.ID); len(es) != 1 || !es[0].Mastered || es[0].Continued != 0 {
		t.Errorf("table %+v: want the run marked mastered, not continued", es)
	}
	h.Press(ebiten.KeyEnter)
	scenetest.Current[*scene.Intro](h) // Storm's
}

// TestCarryOn: Carry on goes to level 9 with no life bonus, and each
// level reached is the run's CONTINUED mark (D62, D63).
func TestCarryOn(t *testing.T) {
	h := bonusHarness(t, config.RushHour)
	before := h.Ctx.Session.Score
	h.Press(ebiten.KeyArrowDown)
	h.Press(ebiten.KeyEnter) // Carry on
	p := scenetest.Current[*Play](h)
	if n := p.World().Level.Number; n != 9 || h.Ctx.Session.Score != before {
		t.Errorf("carried on to level %d with score %d, want 9 and %d", n, h.Ctx.Session.Score, before)
	}
	if s := h.Ctx.Session; !s.Mastered || s.Continued != 9 {
		t.Errorf("session marks %v, %d: want mastered, continued to 9", s.Mastered, s.Continued)
	}
	newLevel(h.Ctx, config.RushHour.Levels, 10)
	if e := entry(h.Ctx.Session); e.Continued != 10 {
		t.Errorf("entry %+v: want continued to 10", e)
	}
}

// TestBonusFlag: -bonus starts games on the mode's bonus level; such a
// game masters no mode (D61).
func TestBonusFlag(t *testing.T) {
	h := scenetest.New(t, 9)
	h.Ctx.NewPlay, h.Ctx.Bonus = New, true
	p := New(h.Ctx, config.Stroll).(*Play)
	if !p.World().Bonus() {
		t.Fatal("-bonus didn't start on the bonus level")
	}
	h.Start(p)
	h.Ticks(scene.FadeTicks)
	endBonus(t, h)
	scenetest.Current[*Mastered](h)
	if h.Ctx.Scores.IsMastered(config.Stroll.ID) {
		t.Error("a -bonus game mastered the mode")
	}
}

// TestFireflyNightReplay plays firefly night from a fixed seed with the
// autopilot at the keys, and logs each firefly's catch and arrival, the
// slips and the nooks, and how it ends (D61). A change to its rules or
// timings shows up here; if it is intended, run the test with -update.
func TestFireflyNightReplay(t *testing.T) {
	h := scenetest.New(t, 11)
	h.Ctx.NewPlay, h.Ctx.Bonus = New, true
	p := New(h.Ctx, config.Stroll).(*Play)
	h.Start(p)
	h.Ticks(scene.FadeTicks)

	keys := map[world.Dir]ebiten.Key{world.Up: ebiten.KeyArrowUp, world.Down: ebiten.KeyArrowDown,
		world.Left: ebiten.KeyArrowLeft, world.Right: ebiten.KeyArrowRight}
	var log strings.Builder
	var a world.Autopilot
	w := p.World()
	caught, on, dying, goals := 0, len(w.Swarm), 0, 0
	for !w.Cleared {
		if key, ok := keys[a.Next(w)]; ok {
			h.Run(scenetest.Hold(1, key))
		} else {
			h.Ticks(1)
		}
		n := 0
		for _, f := range w.Swarm {
			if f.Visiting {
				n++
			}
		}
		switch {
		case w.Stats.Fireflies != caught:
			fmt.Fprintf(&log, "tick %d: firefly %d caught at %d,%d\n", w.Tick, w.Stats.Fireflies, w.Player.Col, w.Player.Row)
		case n > on:
			fmt.Fprintf(&log, "tick %d: a firefly comes in\n", w.Tick)
		case w.Dying > 0 && dying == 0:
			fmt.Fprintf(&log, "tick %d: a slip\n", w.Tick)
		case w.Stats.Goals != goals:
			fmt.Fprintf(&log, "tick %d: a nook\n", w.Tick)
		}
		caught, on, dying, goals = w.Stats.Fireflies, n, w.Dying, w.Stats.Goals
	}
	fmt.Fprintf(&log, "tick %d: ended %v, %d s left\n", w.Tick, p.ending, w.TicksLeft/config.TPS)
	scenetest.Golden(t, "fireflynight", log.String()+summary(p, h.Ctx))
}

// TestLastTrainHomeReplay: the autopilot plays Last Train's bonus level
// from a seed, collecting the fruit in the goods yard and on the river; a
// golden record of what it collects and where (DESIGN §6.7).
func TestLastTrainHomeReplay(t *testing.T) {
	h := scenetest.New(t, 11)
	h.Ctx.NewPlay, h.Ctx.Bonus = New, true
	p := New(h.Ctx, config.LastTrain).(*Play)
	h.Start(p)
	h.Ticks(scene.FadeTicks)

	keys := map[world.Dir]ebiten.Key{world.Up: ebiten.KeyArrowUp, world.Down: ebiten.KeyArrowDown,
		world.Left: ebiten.KeyArrowLeft, world.Right: ebiten.KeyArrowRight}
	var log strings.Builder
	var a world.Autopilot
	w := p.World()
	picked, dying, goals := 0, 0, 0
	for !w.Cleared {
		if key, ok := keys[a.Next(w)]; ok {
			h.Run(scenetest.Hold(1, key))
		} else {
			h.Ticks(1)
		}
		switch {
		case w.Stats.Picked != picked:
			fmt.Fprintf(&log, "tick %d: fruit %d picked at %d,%d\n", w.Tick, w.Stats.Picked, w.Player.Col, w.Player.Row)
		case w.Dying > 0 && dying == 0:
			fmt.Fprintf(&log, "tick %d: a slip\n", w.Tick)
		case w.Stats.Goals != goals:
			fmt.Fprintf(&log, "tick %d: a nook\n", w.Tick)
		}
		picked, dying, goals = w.Stats.Picked, w.Dying, w.Stats.Goals
	}
	fmt.Fprintf(&log, "tick %d: ended %v, %d s left\n", w.Tick, p.ending, w.TicksLeft/config.TPS)
	scenetest.Golden(t, "lasttrainhome", log.String()+summary(p, h.Ctx))
}

// TestTheLastTrainPullsOut: when the clock runs out on the last train
// home, TIME UP holds while the last train pulls out, then the tally
// (DESIGN §6.7).
func TestTheLastTrainPullsOut(t *testing.T) {
	h := scenetest.New(t, 11)
	h.Ctx.NewPlay, h.Ctx.Bonus = New, true
	p := New(h.Ctx, config.LastTrain).(*Play)
	h.Start(p)
	h.Ticks(scene.FadeTicks)
	p.World().TicksLeft = 1
	h.Ticks(1)
	if !p.departing() {
		t.Fatalf("ending %v: want the last train pulling out", p.ending)
	}
	k := p.World().LastTrain()
	x := p.rakeX(k)
	h.Ticks(clearTicks + 1)
	scenetest.Current[*Play](h) // still going
	if p.rakeX(k) <= x {
		t.Errorf("the last train is at %v, from %v: it should be pulling out", p.rakeX(k), x)
	}
	h.Ticks(departTicks - clearTicks + scene.FadeTicks)
	scenetest.Current[*scene.Tally](h)
}
