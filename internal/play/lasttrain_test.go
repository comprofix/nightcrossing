package play

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
)

// lastTrainPlay is a Last Train game on an all-safe level with the combo
// and the day and night cycle.
func lastTrainPlay(t *testing.T) (*scenetest.Harness, *Play) {
	h := scenetest.New(t, 5)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.LastTrain, config.StartLives)
	lv := config.Level{Number: 3, TimerTicks: config.TimerTicks(3), Combo: true, DayCycle: true}
	return h, newPlay(h.Ctx, config.LastTrain.Levels, lv)
}

// TestSkyTurnsWithTheNooks: the level opens in the night before dawn,
// eases to dawn, and moves on a stage only as the nooks fill, one stage
// each DayEaseTicks, never back (D39).
func TestSkyTurnsWithTheNooks(t *testing.T) {
	_, p := lastTrainPlay(t)
	if p.sky != skyBefore {
		t.Fatalf("sky %v at the start, want %v", p.sky, skyBefore)
	}
	ease := func(n int) {
		for range n {
			p.updateSky()
		}
	}
	ease(config.DayEaseTicks)
	if p.stage() != float64(config.Dawn) {
		t.Errorf("sky %v after one ease, want dawn", p.stage())
	}
	ease(5 * config.DayEaseTicks)
	if p.stage() != float64(config.Dawn) {
		t.Errorf("sky %v with no nooks filled, want it to stay at dawn", p.stage())
	}
	for i := range p.world.Filled {
		p.world.Filled[i] = true
	}
	ease(config.DayEaseTicks)
	if p.stage() != float64(config.Day) {
		t.Errorf("sky %v one ease after the nooks filled, want day", p.stage())
	}
	ease(5 * config.DayEaseTicks)
	if p.stage() != float64(config.Night) {
		t.Errorf("sky %v with every nook filled, want night", p.stage())
	}
}

// TestTimesOfDay: day is brighter than dawn and dusk, and the cycle's
// night darker than Stroll's; lamps fade by day (D39).
func TestTimesOfDay(t *testing.T) {
	dawn, day, dusk, nt := skyAt(float64(config.Dawn)), skyAt(float64(config.Day)), skyAt(float64(config.Dusk)), skyAt(float64(config.Night))
	if !(day.level > dawn.level && day.level > dusk.level && dawn.level > nt.level && dusk.level > nt.level) {
		t.Errorf("levels dawn %v day %v dusk %v night %v", dawn.level, day.level, dusk.level, nt.level)
	}
	if nt.level >= night.Level {
		t.Errorf("the cycle's night %v is not darker than Stroll's %v", nt.level, night.Level)
	}
	if day.lamps >= nt.lamps || nt.lamps != 1 {
		t.Errorf("lamps by day %v, by night %v", day.lamps, nt.lamps)
	}
	if mid := skyAt(0.5); mid.level <= dawn.level || mid.level >= day.level {
		t.Errorf("halfway from dawn to day the level is %v", mid.level)
	}
	if skyAt(-1) != nt {
		t.Error("the start is not night")
	}
}

func TestStrollKeepsItsNight(t *testing.T) {
	h := scenetest.New(t, 5)
	p := New(h.Ctx, config.Stroll).(*Play)
	for range 3 * config.DayEaseTicks {
		p.updateSky()
	}
	if p.look() != night || p.lampK() != 1 {
		t.Errorf("Stroll's look %+v, lamps %v: want its fixed night", p.look(), p.lampK())
	}
}

// TestComboPopsUp: a hop the combo multiplies pops its points up over the
// toad, and the pop-up goes after a while (D38).
func TestComboPopsUp(t *testing.T) {
	h, p := lastTrainPlay(t)
	h.Start(p)
	h.Ticks(scene.FadeTicks)
	for range config.ComboStep {
		h.Run(scenetest.Hold(1, ebiten.KeyArrowUp), scenetest.Hold(config.HopTicks))
	}
	if len(p.pops) != 1 || p.pops[0].text != "+20" {
		t.Fatalf("pop-ups %+v, want one +20", p.pops)
	}
	h.Ticks(popTicks)
	if len(p.pops) != 0 {
		t.Errorf("pop-ups %+v still showing", p.pops)
	}
}
