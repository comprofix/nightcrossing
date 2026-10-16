package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The water (DESIGN §3.3: an animated surface, moonlight on the water and
// ripples where platforms meet it), in pixels and ticks.
const (
	waveLen    = 5  // a wave crest on the water
	waveGap    = 19 // between crests along a line
	waveSlowBy = 2  // the surface drifts at this fraction of the row's speed
	waveBob    = 40 // ticks for a crest to rise and fall a pixel

	moonX      = 438 // the middle of the moon's path, from the field's left edge
	moonHalf   = 30  // half the path's width
	moonGlints = 7   // glints in each row of the path
	glintTicks = 9   // a glint moves on after this long
	moonR      = 40  // the moon path's light, in each river row
	moonLight  = 0.35

	wakeLines = 3 // lines trailing behind a platform
	wakeLen   = 7 // the first line's length; each after is shorter
	foamTicks = 6 // the foam at a platform's front changes this often
)

// waveRows are the lines of crests in a river row, from its top, each
// drifting at its own pace.
var waveRows = []struct{ y, phase int }{{4, 0}, {11, 9}, {18, 4}}

// drawWater draws river row r: dark water with crests drifting the way it
// flows, slower than its platforms and bobbing a little, and the moon's
// path, a band of glints that shimmer.
func (p *Play) drawWater(dst *ebiten.Image, r int) {
	w := p.world
	lane := &w.Lanes[r]
	x0, y0 := float32(config.FieldX), float32(cellY(r))
	width := w.Cols * config.Cell
	vector.FillRect(dst, x0, y0, float32(width), config.Cell, gfx.RGBA(resurrect64.Navy), false)

	crest, top := gfx.RGBA(resurrect64.Indigo), gfx.RGBA(resurrect64.Cobalt)
	const period = waveLen + waveGap
	drift := int(w.Tick * int64(lane.Step()) / (config.SubPx * waveSlowBy))
	for i, wr := range waveRows {
		bob := float32(0)
		if (w.Tick/waveBob+int64(i))%2 == 0 {
			bob = 1
		}
		off := ((drift+wr.phase+r*7)%period+period)%period - period
		for dx := off; dx < width; dx += period {
			vector.FillRect(dst, x0+float32(dx), y0+float32(wr.y)+bob, waveLen, 1, crest, false)
			vector.FillRect(dst, x0+float32(dx+1), y0+float32(wr.y)+bob-1, waveLen-2, 1, top, false)
		}
	}

	glint := []int{resurrect64.Sky, resurrect64.Mist, resurrect64.Blue}
	for k := range moonGlints {
		h := noise(w.Tick/glintTicks, int64(r), int64(k))
		gx := moonX - moonHalf + int(h%(2*moonHalf))
		// Glints thin out towards the edges of the path.
		if d := math.Abs(float64(gx-moonX)) / moonHalf; float64(h>>8%100)/100 < d {
			continue
		}
		gy := 2 + int(h>>16%(config.Cell-4))
		gw := 2 + int(h>>24%4)
		vector.FillRect(dst, x0+float32(gx), y0+float32(gy), float32(gw), 1, gfx.RGBA(glint[k%len(glint)]), false)
	}
}

// drawWakes draws the water stirred up by river row r's platforms: foam
// at the front of each and lines trailing behind it. A lily pad that is
// under, or a ship that has gone, stirs nothing.
func (p *Play) drawWakes(dst *ebiten.Image, r int) {
	w := p.world
	lane := &w.Lanes[r]
	y := float32(cellY(r))
	foam := []int{resurrect64.Mist, resurrect64.Fog, resurrect64.Sky}
	wake := gfx.RGBA(resurrect64.Cobalt)
	for i, e := range lane.Entities {
		if st := e.Stage(); st == world.Under || st == world.Sunk {
			continue
		}
		left := float32(config.FieldX) + float32(math.Floor(float64(e.X())/config.SubPx))
		right := left + float32(e.Kind.Length)
		front, back, dir := right, left, float32(1)
		if lane.Dir < 0 {
			front, back, dir = left, right, -1
		}
		// Foam: a broken line just ahead of the front, changing as it goes.
		for py := 6; py < config.Cell-5; py += 2 {
			h := noise(w.Tick/foamTicks, int64(r*100+i), int64(py))
			if h%3 == 0 {
				continue
			}
			fx := front + dir*float32(h>>4%2)
			if dir < 0 {
				fx--
			}
			vector.FillRect(dst, fx, y+float32(py), 1, 1, gfx.RGBA(foam[h>>8%3]), false)
		}
		// Wake: lines trailing behind, each shorter than the one before.
		for k := range wakeLines {
			n := float32(wakeLen - 2*k)
			gap := float32(2 + k*5)
			wx := back - dir*gap
			if dir > 0 {
				wx -= n
			}
			for _, wy := range []float32{7, config.Cell - 8} {
				vector.FillRect(dst, wx, y+wy+float32(k%2), n, 1, wake, false)
			}
		}
	}
}

// noise is a number that looks random from a, b and c but is always the
// same for them, for shimmer and flicker that need no state.
func noise(a, b, c int64) uint32 {
	h := uint32(a)*2654435761 ^ uint32(b)*40503 ^ uint32(c)*2246822519
	h ^= h >> 13
	h *= 1540483477
	h ^= h >> 15
	return h
}
