package play

import (
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/world"
)

// TestGustCues: a gust spell's warning says which way it blows, the wind
// eases in through it, and before each push it builds to full; after the
// spell it dies away (D47).
func TestGustCues(t *testing.T) {
	h := scenetest.New(t, 5)
	h.Ctx.StartLevel = 2
	p := New(h.Ctx, config.Storm).(*Play)
	w := p.World()
	step := func() []world.Event {
		w.TicksLeft = w.Level.TimerTicks // the hero waits on the start row
		p.updateWind()
		return w.Update(world.None)
	}
	until := func(e world.Event, limit int) {
		t.Helper()
		for range limit {
			if slices.Contains(step(), e) {
				return
			}
		}
		t.Fatalf("no %v in %d ticks", e, limit)
	}
	if k, _ := p.windK(); k != 0 {
		t.Fatalf("wind %v before any gust", k)
	}
	until(world.GustWarning, 2*60*config.TPS)
	s := w.Spell(config.Gust)
	if got, want := p.eventMessage(), config.GustMessage(s.Dir); got != want {
		t.Errorf("warning %q, want %q", got, want)
	}
	until(world.GustStart, config.Events[config.Gust].WarnTicks+1)
	calm, dir := p.windK()
	if calm <= 0 || dir != s.Dir {
		t.Errorf("wind %v blowing %d through the spell, want it on, blowing %d", calm, dir, s.Dir)
	}
	for s.NextPush() > 1 {
		step()
	}
	if k, _ := p.windK(); k <= calm {
		t.Errorf("wind %v just before a push, want it stronger than the calm %v", k, calm)
	}
	until(world.GustEnd, s.Timing.Last)
	for range config.OvercastTicks + 1 {
		step()
	}
	if k, _ := p.windK(); k != 0 {
		t.Errorf("wind %v after the spell, want none", k)
	}
}
