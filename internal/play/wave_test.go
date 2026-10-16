package play

import (
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/world"
)

// TestWaveCues: a lane's traffic light is green, goes amber with the hint
// strip's warning as a wave is announced, red through the wave, and green
// again after (D43). Other lanes stay green.
func TestWaveCues(t *testing.T) {
	h := scenetest.New(t, 7)
	p := New(h.Ctx, config.RushHour).(*Play)
	w := p.World()
	run := func(e world.Event, limit int) {
		t.Helper()
		for range limit {
			w.TicksLeft = w.Level.TimerTicks // the hero waits on the start row
			if slices.Contains(w.Update(world.None), e) {
				return
			}
		}
		t.Fatalf("no %v in %d ticks", e, limit)
	}
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.Road && p.trafficFrame(r) != lightGreen {
			t.Fatalf("row %d's light is not green at the start", r)
		}
	}
	run(world.WaveWarning, 60*config.TPS)
	lane := w.Spell(config.Wave).Lane
	if p.trafficFrame(lane) != lightAmber || p.eventMessage() != config.Events[config.Wave].Message {
		t.Errorf("warning: light %d, message %q", p.trafficFrame(lane), p.eventMessage())
	}
	for r := range w.Lanes {
		if r != lane && w.Lanes[r].Kind == config.Road && p.trafficFrame(r) != lightGreen {
			t.Errorf("row %d's light is not green during another lane's wave", r)
		}
	}
	run(world.WaveStart, config.TPS*3)
	if p.trafficFrame(lane) != lightRed || p.eventMessage() != "" {
		t.Errorf("wave: light %d, message %q", p.trafficFrame(lane), p.eventMessage())
	}
	run(world.WaveEnd, 20*config.TPS)
	if p.trafficFrame(lane) != lightGreen {
		t.Errorf("after the wave: light %d", p.trafficFrame(lane))
	}
}
