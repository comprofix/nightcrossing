package play

import (
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
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
