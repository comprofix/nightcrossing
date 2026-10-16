package fx

import (
	"image/color"
	"math/rand/v2"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestParticlesLiveOutTheirLifeAndGo(t *testing.T) {
	ps := NewParticles(rand.New(rand.NewPCG(1, 1)))
	ps.Emit(Burst{X: 10, Y: 10, Count: 20, Speed: 2, Gravity: 0.1, Life: 40, Size: 2,
		Colours: []color.RGBA{{255, 255, 255, 255}}})
	if ps.Len() != 20 {
		t.Fatalf("%d particles, want 20", ps.Len())
	}
	for range 29 { // the shortest life is 3/4 of 40
		ps.Update()
	}
	if ps.Len() != 20 {
		t.Errorf("%d particles left after 29 ticks, want all 20", ps.Len())
	}
	for range 31 {
		ps.Update()
	}
	if ps.Len() != 0 {
		t.Errorf("%d particles left after their longest life", ps.Len())
	}
}

func TestParticlesSameSeedSameMotion(t *testing.T) {
	run := func() []particle {
		ps := NewParticles(rand.New(rand.NewPCG(7, 7)))
		ps.Emit(Burst{Count: 8, Speed: 3, Up: 1, Gravity: 0.2, Drag: 0.9, Life: 60,
			Colours: []color.RGBA{{1, 2, 3, 255}, {4, 5, 6, 255}}})
		for range 25 {
			ps.Update()
		}
		return append([]particle(nil), ps.list...)
	}
	a, b := run(), run()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("particle %d: %+v and %+v from the same seed", i, a[i], b[i])
		}
	}
}

func TestLightTextures(t *testing.T) {
	if v := glowAt(glowSize/2, glowSize/2); v < 0.99 {
		t.Errorf("glow middle %v, want full", v)
	}
	if v := glowAt(0, 0); v != 0 {
		t.Errorf("glow corner %v, want none", v)
	}
	if v := beamAt(1, beamH/2); v < 0.9 {
		t.Errorf("beam at the lamp %v, want nearly full", v)
	}
	if v := beamAt(beamLen/2, 0); v != 0 {
		t.Errorf("beam outside its cone %v, want none", v)
	}
	if beamAt(beamLen/2, beamH/2) <= beamAt(beamLen*3/4, beamH/2) {
		t.Error("the beam does not fade with distance")
	}
}

// TestShaderCompiles checks the lighting shader is valid Kage, so the
// fallback is only for drivers that can't run it.
func TestShaderCompiles(t *testing.T) {
	if _, err := ebiten.NewShader(lightKage); err != nil {
		t.Fatal(err)
	}
}
