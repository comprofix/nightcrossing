package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// riverRow is the river row used by these tests: the one just above the
// verge on level 1's layout.
const riverRow = 5

// riverWorld is a game whose only moving row is a river on riverRow,
// holding one platform p with its left end at x (SubPx units from the
// field's left edge), moving dir at speed units a tick. The rows on either
// side are safe.
func riverWorld(p config.Entity, dives bool, x, dir, speed int) *World {
	lv := testLevel()
	lv.Lanes[riverRow] = config.Lane{Kind: config.River, Dir: dir, Speed: speed,
		Pattern: []config.Spawn{{Entity: p, Gap: 600, Dives: dives}}}
	w := New(session.New(config.Modes[0], config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
	lane := &w.Lanes[riverRow]
	lane.Speed = speed
	lane.Entities = []Entity{{Kind: p, U: x + Buffer, Dives: dives}}
	return w
}

// standOn puts the hero, landed, on riverRow at x (SubPx units).
func standOn(w *World, x int) {
	w.Player = newPlayer(0, riverRow)
	w.Player.fromX, w.Player.toX = x, x
	w.Player.place()
}

func TestHopOntoAPlatformAndRide(t *testing.T) {
	const speed = config.SubPx // a pixel a tick
	// A log right above the hero's start column, which never reaches the
	// edge during the test.
	x0 := (config.StartCol - 1) * cellUnits
	w := riverWorld(config.LogShort, false, x0, 1, speed)
	w.Player = newPlayer(config.StartCol, riverRow+1)

	if ev := run(w, Up, config.HopTicks); slices.Contains(ev, Splashed) {
		t.Fatalf("splashed hopping onto a log: %v", ev)
	}
	if w.Player.Row != riverRow || w.Player.Hopping() {
		t.Fatalf("hero in row %d, hopping %v", w.Player.Row, w.Player.Hopping())
	}
	// Carried from halfway through the hop: half a hop's ticks.
	want := config.StartCol*cellUnits + speed*(config.HopTicks-config.HopTicks/2+1)
	if w.Player.X != want {
		t.Fatalf("landed at %d, want %d", w.Player.X, want)
	}
	// Standing still, it moves with the log, tick for tick.
	for range 60 {
		before, log := w.Player.X, w.Lanes[riverRow].Entities[0].U
		if ev := w.Update(None); len(ev) != 0 {
			t.Fatalf("events %v riding a log", ev)
		}
		if dx, dl := w.Player.X-before, w.Lanes[riverRow].Entities[0].U-log; dx != dl {
			t.Fatalf("the hero moved %d, the log %d", dx, dl)
		}
	}
	if w.Player.Col != columnOf(w.Player.X) {
		t.Errorf("Col %d for X %d", w.Player.Col, w.Player.X)
	}
}

func TestLandingInTheWaterSplashes(t *testing.T) {
	// The only log is far off to the left.
	w := riverWorld(config.LogShort, false, 0, 1, 0)
	w.Player = newPlayer(config.Cols-2, riverRow+1)
	var ev []Event
	for tick := 1; tick <= config.HopTicks; tick++ {
		d := None
		if tick == 1 {
			d = Up
		}
		ev = w.Update(d)
		if tick < config.HopTicks && len(ev) > 0 && ev[0] != Hopped {
			t.Fatalf("tick %d: events %v in the air", tick, ev)
		}
	}
	if !slices.Equal(ev, []Event{Landed, Splashed}) {
		t.Fatalf("landing on water gave %v, want a landing and a splash", ev)
	}
	if w.Player.Death != Splashed || w.Session.Lives != config.StartLives-1 {
		t.Errorf("death %v, lives %d", w.Player.Death, w.Session.Lives)
	}
}

// TestStandGrace lands the hero with its middle just inside and just
// outside the grace past a log's end.
func TestStandGrace(t *testing.T) {
	const x0 = 5 * cellUnits
	end := x0 + config.LogShort.Length*config.SubPx
	grace := config.StandGrace * config.SubPx
	for _, c := range []struct {
		mid   int
		holds bool
	}{
		{x0, true},
		{x0 - grace, true},
		{x0 - grace - 1, false},
		{end + grace - 1, true},
		{end + grace, false},
	} {
		w := riverWorld(config.LogShort, false, x0, 1, 0)
		standOn(w, c.mid-cellUnits/2)
		ev := w.Update(None)
		if splashed := slices.Contains(ev, Splashed); splashed == c.holds {
			t.Errorf("middle at %d (log %d–%d, grace %d): splashed %v", c.mid, x0, end, grace, splashed)
		}
	}
}

// TestCarriedOffTheEdge rides a log to the right edge. The hero is swept
// away on the tick its middle leaves the field.
func TestCarriedOffTheEdge(t *testing.T) {
	const speed = 2 * config.SubPx
	x0 := fieldWidth - config.LogLong.Length*config.SubPx
	w := riverWorld(config.LogLong, false, x0, 1, speed)
	start := fieldWidth - 3*cellUnits
	standOn(w, start)
	// The middle leaves on the tick it reaches fieldWidth.
	want := (fieldWidth - (start + cellUnits/2) + speed - 1) / speed
	for tick := 1; ; tick++ {
		ev := w.Update(None)
		if slices.Contains(ev, Swept) {
			if tick != want {
				t.Errorf("swept on tick %d, want %d", tick, want)
			}
			break
		}
		if len(ev) != 0 || tick > want {
			t.Fatalf("tick %d: events %v, not swept by tick %d", tick, ev, want)
		}
	}
	if w.Player.Death != Swept {
		t.Errorf("death %v, want swept", w.Player.Death)
	}
}

func TestSideHopOffTheEdgeIsIgnored(t *testing.T) {
	w := riverWorld(config.LogLong, false, 0, 1, 0)
	standOn(w, -cellUnits/3) // off the grid, a third over the left edge
	if ev := run(w, Left, config.HopTicks); len(ev) != 0 {
		t.Errorf("events %v from a hop off the edge", ev)
	}
	if ev := run(w, Right, config.HopTicks); !slices.Equal(ev, []Event{Hopped, Landed}) {
		t.Errorf("events %v hopping back along the log", ev)
	}
	if want := -cellUnits/3 + cellUnits; w.Player.X != want {
		t.Errorf("hopped along the log to %d, want %d: hops on the river keep off the grid", w.Player.X, want)
	}
}

// TestHopOffTheRiverLandsOnAColumn rides off the grid, then hops down to
// the verge, landing on the nearest column; the hop moves smoothly there.
func TestHopOffTheRiverLandsOnAColumn(t *testing.T) {
	const speed = 5
	w := riverWorld(config.LogLong, false, 4*cellUnits, 1, speed)
	standOn(w, 6*cellUnits)
	run(w, None, 40) // carried 200 units: 12.5px, past halfway to column 7
	ev := run(w, Down, config.HopTicks)
	if !slices.Equal(ev, []Event{Hopped, Landed}) {
		t.Fatalf("events %v hopping to the verge", ev)
	}
	at(t, w, 7, riverRow+1)
}

func TestDiveCycle(t *testing.T) {
	e := Entity{Dives: true}
	stages := map[Stage]int{}
	prev := Afloat
	var order []Stage
	for range config.DiveCycle {
		s := e.Stage()
		stages[s]++
		if s != prev || len(order) == 0 {
			order = append(order, s)
		}
		prev = s
		e.Phase = (e.Phase + 1) % config.DiveCycle
	}
	if want := []Stage{Afloat, Warning, Sinking, Under, Rising}; !slices.Equal(order, want) {
		t.Errorf("stages %v, want %v", order, want)
	}
	for s, n := range map[Stage]int{Afloat: config.DiveAfloatTicks, Warning: config.DiveWarnTicks,
		Sinking: config.DiveSinkTicks, Under: config.DiveUnderTicks, Rising: config.DiveRiseTicks} {
		if stages[s] != n {
			t.Errorf("stage %d lasts %d ticks, want %d", s, stages[s], n)
		}
	}
	if (Entity{}).Stage() != Afloat {
		t.Error("a platform that doesn't dive is not afloat")
	}
}

// TestADiveTakesTheHeroUnder stands the hero on lily pads just before
// they go under: it is safe through the warning and the sinking, and
// splashes on the first tick under.
func TestADiveTakesTheHeroUnder(t *testing.T) {
	w := riverWorld(config.Pads3, true, 10*cellUnits, 1, 0)
	w.Lanes[riverRow].Entities[0].Phase = config.DiveAfloatTicks - 1 // the warning starts next tick
	standOn(w, 11*cellUnits)
	for tick := 1; ; tick++ {
		ev := w.Update(None)
		if slices.Contains(ev, Splashed) {
			if want := config.DiveWarnTicks + config.DiveSinkTicks + 1; tick != want {
				t.Errorf("went under on tick %d, want %d", tick, want)
			}
			return
		}
		if tick > config.DiveCycle {
			t.Fatal("never went under")
		}
	}
}

func TestDivesAreRandomPerSeedButRepeatable(t *testing.T) {
	phases := func(seed uint64) []int {
		w := New(session.New(config.Modes[0], 3), rich, rand.New(rand.NewPCG(seed, seed)))
		var ps []int
		for _, l := range w.Lanes {
			for _, e := range l.Entities {
				if e.Dives {
					ps = append(ps, e.Phase)
				}
			}
		}
		return ps
	}
	if len(phases(5)) == 0 {
		t.Fatal("level 1 has no diving platforms")
	}
	if !slices.Equal(phases(5), phases(5)) {
		t.Error("one seed gave two sets of dives")
	}
	if slices.Equal(phases(5), phases(6)) {
		t.Error("two seeds gave the same dives")
	}
}

// TestRiverCanBeCrossed plays level 1's river with a careful player: it
// waits on the verge, hops up only when a platform that won't dive soon
// will be under it on landing, and steps back along its platform when it
// nears the edge. It must reach the far bank without a death, for several
// seeds. The full crossing check (D21) is in package config.
func TestRiverCanBeCrossed(t *testing.T) {
	lv := testLevel()
	for r := config.GoalRow + 1; r < config.StartRow; r++ {
		if l := rich.Lanes[r]; l.Kind == config.River {
			lv.Lanes[r] = l
		}
	}
	const verge = riverRow + 1
	for _, sk := range config.Modes {
		for seed := uint64(1); seed <= 8; seed++ {
			w := New(session.New(sk, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
			w.Player = newPlayer(config.StartCol, verge)
			for tick := 0; w.Player.Row != config.GoalRow || w.Player.Hopping(); tick++ {
				if tick > 60*config.TPS {
					t.Fatalf("%s seed %d: still in row %d after a minute", sk.ID, seed, w.Player.Row)
				}
				if ev := w.Update(carefulMove(w)); w.Dying > 0 {
					t.Fatalf("%s seed %d: died (%v) in row %d at tick %d", sk.ID, seed, ev, w.Player.LaneRow(), w.Tick)
				}
			}
		}
	}
}

// carefulMove is the careful player's press for this tick.
func carefulMove(w *World) Dir {
	p := &w.Player
	if p.Hopping() || !w.playing() || p.Row == config.GoalRow {
		return None
	}
	up := p.Row - 1
	switch w.Lanes[up].Kind {
	case config.Bank:
		if n := NookAt(columnOf(p.X)); n >= 0 && !w.Filled[n] {
			return Up
		}
	case config.River:
		if landsSafely(w, up, p.X) && (w.Lanes[up-1].Kind != config.Bank || nookAhead(w, up, p.X)) {
			return Up
		}
	case config.Road:
		if roadClear(w, up, columnOf(p.X)*cellUnits, 3*config.HopTicks) {
			return Up
		}
	default:
		return Up
	}
	lane := &w.Lanes[p.Row]
	if lane.Kind == config.Road && !roadClear(w, p.Row, p.X, 2*config.HopTicks) {
		// Something is coming and the way up is not clear: step back.
		if down := p.Row + 1; w.Lanes[down].Kind != config.Road || roadClear(w, down, p.X, 3*config.HopTicks) {
			return Down
		}
	}
	if lane.Kind != config.River {
		return None
	}
	// On a ship that has started to sink: get off it, back to the row
	// below if need be, or run back towards its stern.
	if sinkingUnder(w) {
		if down := p.Row + 1; w.Lanes[down].Kind != config.River || landsSafely(w, down, p.X) {
			return Down
		}
		if landsSafely(w, p.Row, p.X-lane.Dir*cellUnits) {
			if lane.Dir > 0 {
				return Left
			}
			return Right
		}
	}
	// Below the bank: walk along the platform towards the nearest empty
	// nook.
	if w.Lanes[up].Kind == config.Bank {
		if d := towardsNook(w); d != None {
			dc, _ := d.step()
			if landsSafely(w, p.Row, p.X+dc*cellUnits) {
				return d
			}
		}
	}
	// Nearing the edge it drifts to: step back the other way, if the
	// platform is still under that cell.
	mid, edge := p.Middle(), 2*cellUnits
	if (lane.Dir > 0 && mid > fieldWidth-edge) || (lane.Dir < 0 && mid < edge) {
		back := Left
		if lane.Dir < 0 {
			back = Right
		}
		if landsSafely(w, p.Row, p.X-lane.Dir*cellUnits) {
			return back
		}
		if down := p.Row + 1; w.Lanes[down].Kind != config.River || landsSafely(w, down, p.X) {
			return Down
		}
	}
	return None
}

// landsSafely reports whether a hop starting now that lands in row with
// its left edge at x (before drifting) ends with a platform under the
// hero's middle, with room to spare, on one that will not start to dive
// for a while.
func landsSafely(w *World, row, x int) bool {
	const spare = 6 * config.SubPx
	from, to := &w.Lanes[w.Player.Row], &w.Lanes[row]
	drift := 0
	if from.Kind == config.River {
		drift += from.Step() * (config.HopTicks / 2)
	}
	drift += to.Step() * (config.HopTicks - config.HopTicks/2 + 1)
	mid := x + drift + cellUnits/2
	if mid < cellUnits || mid > fieldWidth-cellUnits {
		return false
	}
	for _, e := range to.Entities {
		if e.Dives && (e.Stage() != Afloat || e.Phase > config.DiveAfloatTicks-2*config.TPS) {
			continue
		}
		if e.Sinks && (e.Stage() != Afloat || e.Wait < 3*config.TPS) {
			continue
		}
		left, right := e.span(0)
		move := to.Step() * (config.HopTicks + 1)
		if e.Heron {
			if hm := e.X() + move + e.HeronCell*cellUnits + cellUnits/2; abs(hm-mid) < cellUnits+spare {
				continue
			}
		}
		if left+move+spare <= mid && mid < right+move-spare {
			return true
		}
	}
	return false
}

// sinkingShip is a world with one barge that sinks, wholly on the field
// and standing still, which starts to sink on the first tick.
func sinkingShip(dir int) *World {
	w := riverWorld(config.Barge, false, 8*cellUnits, dir, 0)
	e := &w.Lanes[riverRow].Entities[0]
	e.Sinks, e.Wait = true, 1
	return w
}

// TestShipSinksBowFirst stands the hero on each piece of a sinking barge,
// in lanes moving both ways. Each piece goes under in turn from the bow,
// the end the ship is heading for; the stern lasts longest, and when it
// goes the ship has sunk.
func TestShipSinksBowFirst(t *testing.T) {
	n := config.Barge.Length / config.Cell
	for _, dir := range []int{1, -1} {
		for piece := range n { // counted from the bow
			w := sinkingShip(dir)
			col := 8 + piece // a piece's cell, from the left
			if dir > 0 {
				col = 8 + n - 1 - piece
			}
			standOn(w, col*cellUnits)
			want := config.SinkWarnTicks + config.SinkBreakTicks + (piece+1)*config.SinkPieceTicks
			for tick := 1; ; tick++ {
				ev := w.Update(None)
				if slices.Contains(ev, Splashed) {
					if tick != want {
						t.Errorf("dir %d, piece %d from the bow: under on tick %d, want %d", dir, piece, tick, want)
					}
					break
				}
				if tick > want {
					t.Fatalf("dir %d, piece %d from the bow: never went under", dir, piece)
				}
			}
			if piece == n-1 {
				if st := w.Lanes[riverRow].Entities[0].Stage(); st != Sunk {
					t.Errorf("dir %d: stage %d once the stern has gone, want sunk", dir, st)
				}
			}
		}
	}
}

func TestShipStagesInOrder(t *testing.T) {
	w := sinkingShip(1)
	var order []Stage
	for range 2 * config.SinkTicks(config.Barge) {
		w.Update(None)
		w.Player.Death, w.Dying = 0, 0 // the hero is not the point here
		if st := w.Lanes[riverRow].Entities[0].Stage(); len(order) == 0 || order[len(order)-1] != st {
			order = append(order, st)
		}
	}
	if want := []Stage{Warning, Breaking, Sinking, Sunk}; !slices.Equal(order, want) {
		t.Errorf("stages %v, want %v", order, want)
	}
}

// TestShipWaitsOnlyOnTheField: a ship's wait runs down only while it is
// wholly on the field, so it never sinks out of sight.
func TestShipWaitsOnlyOnTheField(t *testing.T) {
	const speed = config.SubPx
	// Its front half on the field, moving right, onto it.
	w := riverWorld(config.Barge, false, -config.Barge.Length*config.SubPx/2, 1, speed)
	e := &w.Lanes[riverRow].Entities[0]
	e.Sinks, e.Wait = true, 1000
	half := config.Barge.Length * config.SubPx / 2 / speed
	run(w, None, half-1) // on the last of these, its back is just past the edge
	if e.Wait != 1000 {
		t.Fatalf("wait %d while partly off the field, want 1000", e.Wait)
	}
	run(w, None, 10)
	if e.Wait >= 1000 {
		t.Errorf("wait %d once wholly on, want it running down", e.Wait)
	}
}

// TestSunkShipIsReplaced sinks a ship, then lets it drift off the field: a
// new one comes back in its place, afloat, with a fresh wait.
func TestSunkShipIsReplaced(t *testing.T) {
	const speed = 8 * config.SubPx
	w := riverWorld(config.Barge, false, fieldWidth-config.Barge.Length*config.SubPx-speed, 1, speed)
	e := &w.Lanes[riverRow].Entities[0]
	e.Sinks, e.Sink = true, config.SinkTicks(config.Barge)
	if e.Stage() != Sunk {
		t.Fatal("did not start sunk")
	}
	for range fieldWidth / speed {
		w.Update(None)
		if left, _ := e.span(0); left >= fieldWidth {
			break
		}
	}
	w.Update(None)
	if e.Stage() != Afloat || e.Wait < config.SinkAfterMinTicks || e.Wait > config.SinkAfterMaxTicks {
		t.Errorf("off the field: stage %d, wait %d; want a new ship afloat", e.Stage(), e.Wait)
	}
}

// sinkingUnder reports whether the hero stands on a ship that has started
// to sink.
func sinkingUnder(w *World) bool {
	p := &w.Player
	lane := &w.Lanes[p.Row]
	for _, e := range lane.Entities {
		if left, right := e.span(-config.StandGrace); e.Sinks && e.Stage() != Afloat && left <= p.Middle() && p.Middle() < right {
			return true
		}
	}
	return false
}

// roadClear reports whether no vehicle in road row will touch a hero at x
// over the next ticks, with a few pixels to spare.
func roadClear(w *World, row, x, ticks int) bool {
	const spare = 4 * config.SubPx
	l := &w.Lanes[row]
	m := config.HeroMargin * config.SubPx
	left, right := x+m-spare, x+cellUnits-m+spare
	for t := 0; t <= ticks; t++ {
		for _, e := range l.Entities {
			el, er := e.span(config.HitMargin)
			if el+l.Step()*t < right && left < er+l.Step()*t {
				return false
			}
		}
	}
	return true
}

// towardsNook is the way along the row to the nearest empty nook, or None
// when the hero is below one.
func towardsNook(w *World) Dir {
	col := columnOf(w.Player.X)
	best, dir := config.Cols, None
	for i, c := range config.Nooks {
		if w.Filled[i] {
			continue
		}
		for dc := range config.NookWidth {
			d := c + dc - col
			if d == 0 {
				return None
			}
			if abs(d) < best {
				best, dir = abs(d), Right
				if d < 0 {
					dir = Left
				}
			}
		}
	}
	return dir
}

// nookAhead reports whether an empty nook is still to come for a hero
// landing at x in river row, which flows one way: it can be carried to it.
func nookAhead(w *World, row, x int) bool {
	dir := w.Lanes[row].Dir
	for i, c := range config.Nooks {
		if w.Filled[i] {
			continue
		}
		left, right := c*cellUnits, (c+config.NookWidth)*cellUnits
		if (dir > 0 && right > x+cellUnits) || (dir < 0 && left < x) {
			return true
		}
	}
	return false
}
