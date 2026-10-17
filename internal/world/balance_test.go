package world

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

var balance = flag.Bool("balance", false, "run TestStormBalance, which prints deaths per storm, wet and dry (D52)")

// balanceRun is what the autopilot made of one kind of storm on a level:
// the storms begun, the deaths while one was on, and the nooks filled and
// ticks played in a storm and in the clear.
type balanceRun struct {
	storms                 int
	stormDeaths            int
	stormNooks, clearNooks int
	stormTicks, clearTicks int
}

func (r *balanceRun) add(o balanceRun) {
	r.storms += o.storms
	r.stormDeaths += o.stormDeaths
	r.stormNooks += o.stormNooks
	r.clearNooks += o.clearNooks
	r.stormTicks += o.stormTicks
	r.clearTicks += o.clearTicks
}

// playStorms has the autopilot play level n of Storm for minutes with
// every storm kind k. Its lives are kept up, and each time it fills the
// last nook they empty again, so the level and its storms run on.
func playStorms(n int, k config.StormKind, seed uint64, minutes int) balanceRun {
	lv := config.Storm.Levels.Level(n)
	lv.Storms = k
	w := New(session.New(config.Storm, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
	var a Autopilot
	var out balanceRun
	for range minutes * 60 * config.TPS {
		if w.Cleared {
			w.Filled, w.Cleared = [len(config.Nooks)]bool{}, false
			w.newAttempt()
		}
		on := w.Spell(config.StormFront).Phase == SpellOn
		if on {
			out.stormTicks++
		} else {
			out.clearTicks++
		}
		for _, e := range w.Update(a.Next(w)) {
			switch e {
			case StormStart:
				out.storms++
			case Goal:
				if on {
					out.stormNooks++
				} else {
					out.clearNooks++
				}
			case Squashed, Splashed, Swept, Caught, Struck, TimeUp:
				if on {
					out.stormDeaths++
				}
			}
		}
		w.Session.Lives = config.StartLives
	}
	return out
}

// TestStormBalance has the autopilot play each of Storm's levels with
// only wet storms, then only dry ones, and prints for each kind its deaths
// per storm and how fast it fills nooks in a storm against in the clear,
// to compare the two (D52). It runs only with -balance:
//
//	go test ./internal/world -run TestStormBalance -balance -v
//
// It never fails. The autopilot is a rough player that looks ahead, so it
// sees every push and flood coming and seldom dies; how much a storm slows
// it is the better measure. Balance is judged by play: the numbers only
// flag a big difference between the kinds.
func TestStormBalance(t *testing.T) {
	if !*balance {
		t.Skip("run with -balance to print deaths per storm, wet and dry (D52)")
	}
	const seeds, minutes = 3, 10
	kinds := []config.StormKind{config.WetStorm, config.DryStorm}
	var results [config.EventTopLevel + 1][3]balanceRun
	var wg sync.WaitGroup
	var mu sync.Mutex
	for n := 1; n <= config.EventTopLevel; n++ {
		for _, k := range kinds {
			for seed := uint64(1); seed <= seeds; seed++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					r := playStorms(n, k, seed, minutes)
					mu.Lock()
					defer mu.Unlock()
					results[n][k].add(r)
				}()
			}
		}
	}
	wg.Wait()
	perMin := func(nooks, ticks int) float64 { return float64(nooks) * 60 * config.TPS / float64(max(ticks, 1)) }
	t.Logf("%d seeds, %d minutes each. Nooks a minute in a storm, and in the clear; deaths per storm", seeds, minutes)
	t.Logf("level |  wet: storms deaths/storm storm clear |  dry: storms deaths/storm storm clear")
	for n := 1; n <= config.EventTopLevel; n++ {
		line := ""
		for _, k := range kinds {
			r := results[n][k]
			line += fmt.Sprintf(" | %10d %12.2f %5.1f %5.1f", r.storms, float64(r.stormDeaths)/float64(max(r.storms, 1)),
				perMin(r.stormNooks, r.stormTicks), perMin(r.clearNooks, r.clearTicks))
		}
		t.Logf("%5d%s", n, line)
	}
}
