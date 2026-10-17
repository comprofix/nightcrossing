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

// TestStormSky: Storm is overcast from its first tick, lamps and all, and
// stays so; the modes before it aren't (D49).
func TestStormSky(t *testing.T) {
	h := scenetest.New(t, 3)
	p := New(h.Ctx, config.Storm).(*Play)
	for range 30 * config.TPS {
		if p.overcastK() != 1 || p.lampK() != 1 {
			t.Fatalf("tick %d: overcast %v lamps %v, want both full", p.ticks, p.overcastK(), p.lampK())
		}
		p.world.TicksLeft = p.world.Level.TimerTicks
		p.updateEffects()
		p.world.Update(world.None)
		p.ticks++
	}
	if q := New(scenetest.New(t, 3).Ctx, config.RushHour).(*Play); q.overcastK() != 0 {
		t.Errorf("Rush Hour starts overcast %v", q.overcastK())
	}
}

// TestLightning: under the storm sky lightning flashes every 15–30 s,
// lifting the light toward daylight and fading, with thunder a beat later;
// never while an event is announced; softer with reduced motion (D49).
// Its timing takes nothing from the game's random numbers, so the replay
// in road_storm.golden plays the same with or without it.
func TestLightning(t *testing.T) {
	h := scenetest.New(t, 9)
	h.Ctx.StartLevel = 3
	p := New(h.Ctx, config.Storm).(*Play)
	var flashes, thunders []int
	for tick := range 3 * 60 * config.TPS {
		p.world.TicksLeft = p.world.Level.TimerTicks
		p.updateEffects()
		if p.flash == 0 {
			flashes = append(flashes, tick)
			for _, s := range p.world.Spells {
				if s.Phase == world.SpellWarning {
					t.Errorf("tick %d: a flash during an event's warning", tick)
				}
			}
			before := p.look()
			if hard, soft := p.lightning(before, false), p.lightning(before, true); hard.Level <= before.Level || soft.Level > hard.Level {
				t.Errorf("tick %d: light %v, flash %v, soft %v", tick, before.Level, hard.Level, soft.Level)
			}
		}
		if p.thunderNow() {
			thunders = append(thunders, tick)
		}
		p.world.Update(world.None)
	}
	if len(flashes) < 3*60/30 || len(flashes) > 3*60/15+1 {
		t.Errorf("%d flashes in three minutes: %v", len(flashes), flashes)
	}
	for i := 1; i < len(flashes); i++ {
		if gap := flashes[i] - flashes[i-1]; gap < lightningMin {
			t.Errorf("flashes %d ticks apart", gap)
		}
	}
	if len(thunders) != len(flashes) || len(thunders) > 0 && thunders[0] != flashes[0]+thunderDelay {
		t.Errorf("thunder at %v for flashes at %v", thunders, flashes)
	}
	p.flash = flashHold + flashFade
	if look := p.look(); p.lightning(look, false) != look {
		t.Error("the light is lifted after the flash has faded")
	}
}
