package play

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// pickupsPlay is a Stroll game on an all-safe level with the far bank at
// the top and fruit, faded in and ready.
func pickupsPlay(t *testing.T) (*scenetest.Harness, *Play) {
	h := scenetest.New(t, 5)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.Stroll, config.StartLives)
	lv := config.Level{Number: 2, TimerTicks: config.TimerTicks(2), FruitPts: config.PointsFruitFirst, Fruit: 1}
	lv.Lanes[config.GoalRow].Kind = config.Bank
	p := newPlay(h.Ctx, config.StrollLevels, lv)
	h.Start(p)
	h.Ticks(scene.FadeTicks + 1)
	return h, p
}

// TestOneUpPopsUp: a life won pops up "1UP" over the toad (D57).
func TestOneUpPopsUp(t *testing.T) {
	h, p := pickupsPlay(t)
	h.Ctx.Session.Score = config.ExtraLifeEvery - config.PointsHop
	hop(h)
	if h.Ctx.Session.Lives != config.StartLives+1 {
		t.Fatalf("%d lives, want a life from the points", h.Ctx.Session.Lives)
	}
	if !slices.ContainsFunc(p.pops, func(pp pop) bool { return pp.text == "1UP" }) {
		t.Errorf("pop-ups %+v: want a 1UP", p.pops)
	}
}

// TestTallyLines: the tally has a FRUIT line on a level with fruit, and a
// FIREFLY BONUS line only when a firefly was caught at full lives (D57,
// D59).
func TestTallyLines(t *testing.T) {
	_, p := pickupsPlay(t)
	labels := func() []string {
		var out []string
		for _, l := range p.tallyLines() {
			out = append(out, l.Label)
		}
		return out
	}
	if got := labels(); !slices.Contains(got, "FRUIT") || slices.Contains(got, "FIREFLY BONUS") {
		t.Errorf("lines %v: want FRUIT and no FIREFLY BONUS", got)
	}
	p.world.Stats.AtCap = config.PointsFireflyAtCap
	if got := labels(); !slices.Contains(got, "FIREFLY BONUS") {
		t.Errorf("lines %v: want FIREFLY BONUS after one at full lives", got)
	}
	p.world.Level.FruitPts = 0
	if got := labels(); slices.Contains(got, "FRUIT") {
		t.Errorf("lines %v: a FRUIT line on a level without fruit", got)
	}
}

// TestPickupsReplay plays Stroll's level 8 from a fixed seed with the
// autopilot at the keys, and logs the fruit and the fireflies as they come
// and go: each fruit shown and taken, each firefly's visit, its landings
// and its catch, and the lives (D57–D59). A change to their rules or
// timings shows up here; if it is intended, run the test with -update.
func TestPickupsReplay(t *testing.T) {
	h := scenetest.New(t, 7)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.Stroll, config.StartLives)
	p := newPlay(h.Ctx, config.StrollLevels, config.StrollLevels.Level(8))
	h.Start(p)
	h.Ticks(scene.FadeTicks)

	keys := map[world.Dir]ebiten.Key{world.Up: ebiten.KeyArrowUp, world.Down: ebiten.KeyArrowDown,
		world.Left: ebiten.KeyArrowLeft, world.Right: ebiten.KeyArrowRight}
	var log strings.Builder
	var a world.Autopilot
	w := p.World()
	nook, visiting, landed, fruit, caught := -1, false, false, 0, 0
	for range 90 * config.TPS {
		if key, ok := keys[a.Next(w)]; ok {
			h.Run(scenetest.Hold(1, key))
		} else {
			h.Ticks(1)
		}
		if w = p.World(); w.Over || w.Level.Number != 8 {
			break
		}
		f := w.Flyer
		switch {
		case w.FruitNook != nook && w.FruitNook >= 0:
			fmt.Fprintf(&log, "tick %d: fruit in nook %d\n", w.Tick, w.FruitNook)
		case w.Stats.Fruit != fruit:
			fmt.Fprintf(&log, "tick %d: fruit taken, %d points\n", w.Tick, w.Stats.Fruit-fruit)
		case w.FruitNook != nook:
			fmt.Fprintf(&log, "tick %d: fruit gone\n", w.Tick)
		}
		switch {
		case w.Stats.Fireflies != caught:
			fmt.Fprintf(&log, "tick %d: firefly caught, %d lives\n", w.Tick, h.Ctx.Session.Lives)
		case f.Visiting && !visiting:
			fmt.Fprintf(&log, "tick %d: firefly comes in at x %d\n", w.Tick, f.X/config.SubPx)
		case !f.Visiting && visiting:
			fmt.Fprintf(&log, "tick %d: firefly gone\n", w.Tick)
		case f.Landed && !landed:
			fmt.Fprintf(&log, "tick %d: firefly lands in row %d at x %d\n", w.Tick, f.Y/config.Cell/config.SubPx, f.X/config.SubPx)
		}
		nook, visiting, landed, fruit, caught = w.FruitNook, f.Visiting, f.Landed, w.Stats.Fruit, w.Stats.Fireflies
	}
	if !strings.Contains(log.String(), "firefly lands") || !strings.Contains(log.String(), "fruit in nook") {
		t.Errorf("no firefly landed or no fruit showed:\n%s", log.String())
	}
	scenetest.Golden(t, "pickups", log.String()+summary(p, h.Ctx))
}
