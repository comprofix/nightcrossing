package play

import (
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/world"
)

// weatherPlay is Rush Hour's level 3 by day, with weather k forced on.
func weatherPlay(t *testing.T, k config.EventKind) *Play {
	h := scenetest.New(t, 2)
	h.Ctx.StartLevel = 3
	p := New(h.Ctx, config.RushHour).(*Play)
	p.world.Filled[0] = true // day
	for range 3 * config.DayEaseTicks {
		p.updateSky()
	}
	s := p.world.Spell(k)
	s.Phase, s.Left = world.SpellWarning, config.Events[k].WarnTicks
	return p
}

// TestOvercastLightsTheDay: through fog's warning the light dims and greys,
// the lamps and headlights come on at full, and the street lamps come on
// one by one; after it, all goes back to the day (D44).
func TestOvercastLightsTheDay(t *testing.T) {
	p := weatherPlay(t, config.Fog)
	day, dayLamps := p.look(), p.lampK()
	if dayLamps >= 0.5 {
		t.Fatalf("lamps %v by day", dayLamps)
	}
	var on []int // how many street lamps are on at each step
	for range config.OvercastTicks {
		p.updateWeather()
		n := 0
		for i := range lamps {
			if p.streetLampK(i) == 1 {
				n++
			}
		}
		on = append(on, n)
	}
	if p.lampK() != 1 || p.fogK() != 1 || p.rainK() != 0 {
		t.Errorf("after the warning: lamps %v, fog %v, rain %v", p.lampK(), p.fogK(), p.rainK())
	}
	if l := p.look(); l.Level >= day.Level || l.Ambient == day.Ambient {
		t.Errorf("overcast look %+v, the day's %+v", l, day)
	}
	for i := 1; i < len(on); i++ {
		if on[i] < on[i-1] || on[i]-on[i-1] > 1 {
			t.Fatalf("street lamps on, step by step: %v", on)
		}
	}
	if on[0] != 0 || on[len(on)-1] != len(lamps) {
		t.Errorf("street lamps on, step by step: %v", on)
	}
	s := p.world.Spell(config.Fog)
	s.Phase = world.SpellQuiet
	for range config.OvercastTicks {
		p.updateWeather()
	}
	if p.look() != day || p.lampK() != dayLamps {
		t.Errorf("after the fog: %+v lamps %v, want the day's", p.look(), p.lampK())
	}
}

func TestRainOnlyRains(t *testing.T) {
	p := weatherPlay(t, config.Rain)
	for range config.OvercastTicks {
		p.updateWeather()
	}
	if p.rainK() != 1 || p.fogK() != 0 || p.overcastK() != 1 {
		t.Errorf("rain %v, fog %v, overcast %v", p.rainK(), p.fogK(), p.overcastK())
	}
}

// TestFogPatchesStayOnTheRoad: every patch is over the road rows.
func TestFogPatchesStayOnTheRoad(t *testing.T) {
	p := weatherPlay(t, config.Fog)
	for tick := range 2000 {
		p.ticks = tick
		for i := range fogPatches {
			_, y := p.fogPatch(i)
			if y < cellY(8)-fogPatchH/2 || y+fogPatchH > cellY(12)+fogPatchH/2 {
				t.Fatalf("tick %d: patch %d at y %v, off the road", tick, i, y)
			}
		}
	}
}
