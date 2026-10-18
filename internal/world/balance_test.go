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

var balance = flag.Bool("balance", false, "run the balance reports: TestStormBalance (deaths per storm, wet and dry, D52) and TestLivesBalance (lives level by level, D60)")

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
			case Squashed, Splashed, Swept, Caught, Snatched, Nipped, Bitten, Struck, TimeUp:
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

// livesLevel is what the autopilot made of one level of a game: the
// deaths, the fireflies caught, the lives won from points, the times its
// lives had to be topped up to keep the game going, and its lives at the
// end of the level.
type livesLevel struct {
	deaths, fireflies, fromPoints, topUps, lives int
}

// playLives has the autopilot play a game of mode m from level 1 to level
// last, as a player would: lives carry from level to level, and the score
// earns them. If the last one goes, a life is put back so the game goes
// on, and counted as a top-up. A level not cleared in minutes is left.
func playLives(m config.Mode, last int, seed uint64, minutes int) []livesLevel {
	s := session.New(m, config.StartLives)
	rng := rand.New(rand.NewPCG(seed, seed))
	var a Autopilot
	out := make([]livesLevel, last+1)
	for n := 1; n <= last; n++ {
		w := New(s, m.Levels.Level(n), rng)
		r := &out[n]
		for t := 0; t < minutes*60*config.TPS && !w.Cleared; t++ {
			caught := false
			for _, e := range w.Update(a.Next(w)) {
				switch e {
				case Squashed, Splashed, Swept, Caught, Snatched, Nipped, Bitten, Bumped, Struck, TimeUp:
					r.deaths++
				case FireflyCaught:
					r.fireflies++
					caught = true
				case ExtraLife:
					if !caught {
						r.fromPoints++
					}
				}
			}
			if s.Lives == 0 && w.Dying > 0 {
				s.Lives = 1
				r.topUps++
			}
		}
		r.lives = s.Lives
	}
	return out
}

// TestLivesBalance has the autopilot play each mode from level 1 through
// the first loops, as a player would, and prints for each level the
// deaths, fireflies caught, lives from points and the lives left, against
// D60's target: lives about even by level 6, then a slow loss of 1–2 a
// level. It runs only with -balance:
//
//	go test ./internal/world -run TestLivesBalance -balance -v
//
// It never fails. The autopilot dies less than a person and takes every
// firefly it can reach, so the numbers are a guide; balance is judged by
// play.
func TestLivesBalance(t *testing.T) {
	if !*balance {
		t.Skip("run with -balance to print lives level by level (D60)")
	}
	const seeds, minutes = 4, 5
	last := 8 + 2*config.LoopLevels
	for _, m := range config.Modes {
		results := make([][]livesLevel, seeds)
		var wg sync.WaitGroup
		for i := range seeds {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = playLives(m, last, uint64(i+1), minutes)
			}()
		}
		wg.Wait()
		t.Logf("%s: %d games, averages per game", m.Name, seeds)
		t.Logf("level | deaths fireflies points  net | lives top-ups")
		for n := 1; n <= last; n++ {
			var sum livesLevel
			for _, r := range results {
				l := r[n]
				sum.deaths += l.deaths
				sum.fireflies += l.fireflies
				sum.fromPoints += l.fromPoints
				sum.topUps += l.topUps
				sum.lives += l.lives
			}
			avg := func(v int) float64 { return float64(v) / seeds }
			net := avg(sum.fireflies + sum.fromPoints - sum.deaths)
			t.Logf("%5d | %6.1f %9.1f %6.1f %+4.1f | %5.1f %7.1f", n, avg(sum.deaths), avg(sum.fireflies),
				avg(sum.fromPoints), net, avg(sum.lives), avg(sum.topUps))
		}
	}
}
