package fx

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Burst is a puff of particles from one point (DESIGN §3.3: splashes,
// sparks and dust puffs).
type Burst struct {
	X, Y    float64      // where, in pixels
	Count   int          // how many
	Speed   float64      // the fastest start speed, pixels per tick
	Up      float64      // added upward speed, pixels per tick
	Gravity float64      // pull down each tick
	Drag    float64      // speed kept each tick, 0–1; 0 means 1
	Life    int          // ticks each lasts, give or take a quarter
	Size    int          // pixels square
	Colours []color.RGBA // each particle takes one at random
	Streak  bool         // drawn smooth, a streak along its path, as a spark is
}

// particle is one moving dot.
type particle struct {
	x, y, vx, vy  float64
	gravity, drag float64
	age, life     int
	size          int
	colour        color.RGBA
	streak        bool
}

// Particles is a set of moving dots. Update moves them each tick and Draw
// draws them; Draw changes nothing.
type Particles struct {
	list []particle
	rng  *rand.Rand
}

// NewParticles makes an empty set whose random choices come from rng, so a
// game stays the same from its seed.
func NewParticles(rng *rand.Rand) *Particles { return &Particles{rng: rng} }

// Len is how many particles are alive.
func (ps *Particles) Len() int { return len(ps.list) }

// Emit adds a burst.
func (ps *Particles) Emit(b Burst) {
	drag := b.Drag
	if drag == 0 {
		drag = 1
	}
	for range b.Count {
		a := ps.rng.Float64() * 2 * math.Pi
		s := b.Speed * (0.3 + 0.7*ps.rng.Float64())
		life := b.Life * (3 + ps.rng.IntN(3)) / 4
		ps.list = append(ps.list, particle{
			x: b.X, y: b.Y,
			vx: s * math.Cos(a), vy: s*math.Sin(a) - b.Up,
			gravity: b.Gravity, drag: drag,
			life: max(life, 1), size: max(b.Size, 1),
			colour: b.Colours[ps.rng.IntN(len(b.Colours))],
			streak: b.Streak,
		})
	}
}

// Update moves every particle one tick and drops the ones that have run out.
func (ps *Particles) Update() {
	live := ps.list[:0]
	for _, p := range ps.list {
		p.age++
		if p.age >= p.life {
			continue
		}
		p.vx *= p.drag
		p.vy = p.vy*p.drag + p.gravity
		p.x += p.vx
		p.y += p.vy
		live = append(live, p)
	}
	ps.list = live
}

// Clear removes every particle.
func (ps *Particles) Clear() { ps.list = ps.list[:0] }

// Draw draws the particles onto dst k times their own size, offset by ox,
// oy in dst's pixels, on whole pixels of dst. A particle shrinks by one of
// its own pixels over the last part of its life.
func (ps *Particles) Draw(dst *ebiten.Image, ox, oy, k float64) {
	for _, p := range ps.list {
		size := p.size
		if size > 1 && p.age*4 >= p.life*3 {
			size--
		}
		x := float32(math.Floor(ox + k*(p.x-float64(size)/2)))
		y := float32(math.Floor(oy + k*(p.y-float64(size)/2)))
		s := float32(k * float64(size))
		vector.FillRect(dst, x, y, s, s, p.colour, false)
	}
}

// streakBack is how many ticks of its path a streak shows behind it.
const streakBack = 1.5

// DrawSmooth draws the particles as Draw does, smoothed at dst's own
// pixels: round drops and motes, sparks as streaks along their paths. A
// particle shrinks to half its size over the last part of its life.
func (ps *Particles) DrawSmooth(dst *ebiten.Image, ox, oy, k float64) {
	for _, p := range ps.list {
		size := float64(p.size)
		if late := float64(p.age)/float64(p.life) - 0.75; late > 0 {
			size *= 1 - 2*late
		}
		x, y := float32(ox+k*p.x), float32(oy+k*p.y)
		if p.streak {
			bx, by := float32(k*p.vx*streakBack), float32(k*p.vy*streakBack)
			vector.StrokeLine(dst, x-bx, y-by, x, y, float32(k*size*0.8), p.colour, true)
			vector.FillCircle(dst, x, y, float32(k*size*0.4), p.colour, true)
			continue
		}
		vector.FillCircle(dst, x, y, float32(k*size/2), p.colour, true)
	}
}
