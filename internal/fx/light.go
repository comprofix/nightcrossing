// Package fx draws Night Crossing's effects on top of the scene: the night
// lighting and particles (DESIGN §3.3). It only draws; the play scene
// decides what lights and particles there are from the world's state and
// events.
package fx

import (
	_ "embed"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed light.kage
var lightKage []byte

//go:embed bright.kage
var brightKage []byte

// Shape is the shape of a light.
type Shape int

// The shapes.
const (
	Glow Shape = iota // round, brightest in the middle
	Beam              // a cone from X, Y facing right (or left with Flip), like a headlight
)

// Light is one light source, in the lit image's pixels.
type Light struct {
	Shape    Shape
	X, Y     float64
	Radius   float64 // how far it reaches
	Colour   color.RGBA
	Strength float64 // 0–1 and a little over
	Flip     bool    // a beam facing left
}

// Look is how the night looks: the light everywhere and how sources add to
// it.
type Look struct {
	Ambient  color.RGBA // the light everywhere
	Level    float64    // how much of Ambient (0–1)
	Max      float64    // the most light a pixel can take
	Bands    int        // light steps, for a pixel-art look; 0 is smooth
	Glow     float64    // how strongly bright parts glow onto what is round them; 0 is none
	GlowFrom float64    // the brightness where the glow starts (0–1)
}

// Lighting lights a scene: draw the scene into Scene, add lights, then Draw
// the lit result. With shaders it combines them in a Kage shader; without,
// it multiplies the light map onto the scene with a blend, which gives the
// same picture but can't band the light or lift colours above their own
// value (PLAN §3).
type Lighting struct {
	w, h      int
	useShader bool

	// Made on first use, since images can only be made once the game is
	// running. They are drawing buffers, not game state.
	scene, lights, out *ebiten.Image
	shader             *ebiten.Shader
	glow, beam         *ebiten.Image
	// The glow: the bright parts, then smaller and smaller copies of them,
	// which blur them as they are scaled back up.
	bright       *ebiten.Image
	blur         [3]*ebiten.Image
	brightShader *ebiten.Shader
	look         Look
}

// NewLighting lights scenes w×h pixels. useShader false always takes the
// fallback; a shader that fails to compile does too.
func NewLighting(w, h int, useShader bool) *Lighting {
	return &Lighting{w: w, h: h, useShader: useShader}
}

// Size of the light textures, in pixels.
const (
	glowSize = 64
	beamLen  = 64
	beamH    = 48
)

func (l *Lighting) ensure() {
	if l.scene != nil {
		return
	}
	l.scene = ebiten.NewImage(l.w, l.h)
	l.lights = ebiten.NewImage(l.w, l.h)
	l.out = ebiten.NewImage(l.w, l.h)
	l.glow = texture(glowSize, glowSize, glowAt)
	l.beam = texture(beamLen, beamH, beamAt)
	if l.useShader {
		s, err := ebiten.NewShader(lightKage)
		if err != nil {
			log.Printf("lighting shader: %v; using the fallback", err)
		}
		l.shader = s
		if b, err := ebiten.NewShader(brightKage); err == nil {
			l.brightShader = b
		} else {
			log.Printf("glow shader: %v; no glow", err)
		}
		l.bright = ebiten.NewImage(l.w, l.h)
		for i := range l.blur {
			l.blur[i] = ebiten.NewImage(max(l.w>>(i+1), 1), max(l.h>>(i+1), 1))
		}
	}
}

// Shaded reports whether the shader is in use, once lighting has begun.
func (l *Lighting) Shaded() bool { return l.shader != nil }

// Begin starts a frame with the given look. It returns the scene image to
// draw into, cleared.
func (l *Lighting) Begin(look Look) *ebiten.Image {
	l.ensure()
	l.look = look
	l.scene.Clear()
	if l.shader != nil {
		l.lights.Clear()
	} else {
		l.lights.Fill(scale(look.Ambient, look.Level))
	}
	return l.scene
}

// Add adds a light to the light map.
func (l *Lighting) Add(li Light) {
	if li.Strength <= 0 || li.Radius <= 0 {
		return
	}
	var op ebiten.DrawImageOptions
	op.Blend = ebiten.BlendLighter
	op.Filter = ebiten.FilterLinear
	switch li.Shape {
	case Glow:
		k := 2 * li.Radius / glowSize
		op.GeoM.Translate(-glowSize/2, -glowSize/2)
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(li.X, li.Y)
		op.ColorScale.ScaleWithColor(scale(li.Colour, li.Strength))
		l.lights.DrawImage(l.glow, &op)
	case Beam:
		k := li.Radius / beamLen
		op.GeoM.Translate(0, -beamH/2)
		if li.Flip {
			op.GeoM.Scale(-1, 1)
		}
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(li.X, li.Y)
		op.ColorScale.ScaleWithColor(scale(li.Colour, li.Strength))
		l.lights.DrawImage(l.beam, &op)
	}
}

// Draw draws the lit scene onto dst at x, y.
func (l *Lighting) Draw(dst *ebiten.Image, x, y float64) {
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(x, y)
	if l.shader != nil {
		a := l.look.Ambient
		sop := &ebiten.DrawRectShaderOptions{}
		sop.Images[0] = l.scene
		sop.Images[1] = l.lights
		sop.Uniforms = map[string]any{
			"Ambient": []float32{
				float32(float64(a.R) / 255 * l.look.Level),
				float32(float64(a.G) / 255 * l.look.Level),
				float32(float64(a.B) / 255 * l.look.Level),
			},
			"Max":   float32(l.look.Max),
			"Bands": float32(l.look.Bands),
		}
		l.out.Clear()
		l.out.DrawRectShader(l.w, l.h, l.shader, sop)
		if l.look.Glow > 0 && l.brightShader != nil {
			l.addGlow()
		}
		dst.DrawImage(l.out, &op)
		return
	}
	// Fallback: scene × light map, by a multiplying blend.
	l.out.Clear()
	l.out.DrawImage(l.scene, nil)
	l.out.DrawImage(l.lights, &ebiten.DrawImageOptions{Blend: ebiten.Blend{
		BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
		BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
		BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
		BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
		BlendOperationRGB:           ebiten.BlendOperationAdd,
		BlendOperationAlpha:         ebiten.BlendOperationAdd,
	}})
	dst.DrawImage(l.out, &op)
}

// scale is c at strength k, as an opaque colour.
func scale(c color.RGBA, k float64) color.RGBA {
	f := func(v uint8) uint8 { return uint8(min(math.Round(float64(v)*k), 255)) }
	return color.RGBA{f(c.R), f(c.G), f(c.B), 255}
}

// texture makes a w×h white light texture whose brightness at each pixel is
// at(x, y), from 0 to 1.
func texture(w, h int, at func(x, y float64) float64) *ebiten.Image {
	pix := make([]byte, 4*w*h)
	for y := range h {
		for x := range w {
			v := byte(math.Round(255 * min(max(at(float64(x)+0.5, float64(y)+0.5), 0), 1)))
			i := 4 * (y*w + x)
			pix[i], pix[i+1], pix[i+2], pix[i+3] = v, v, v, 255
		}
	}
	img := ebiten.NewImage(w, h)
	img.WritePixels(pix)
	return img
}

// glowAt is a round light, full in the middle and gone at the edge.
func glowAt(x, y float64) float64 {
	dx, dy := (x-glowSize/2)/(glowSize/2), (y-glowSize/2)/(glowSize/2)
	return falloff(math.Sqrt(dx*dx + dy*dy))
}

// beamAt is a cone opening to the right from the middle of the left edge,
// fading with distance and soft at its sides.
func beamAt(x, y float64) float64 {
	d := x / beamLen
	half := 3 + d*(beamH/2-3) // the cone's half width here, in pixels
	side := math.Abs(y-beamH/2) / half
	return falloff(d) * min(max((1-side)*3, 0), 1)
}

// falloff is the light at distance d from 0 (full) to 1 (none).
func falloff(d float64) float64 {
	if d >= 1 {
		return 0
	}
	v := 1 - d*d
	return v * v
}

// addGlow spreads the bright parts of the lit scene onto what is round
// them (DESIGN §3.3: a light glow around bright sources). The bright parts
// are halved in size three times, which blurs them, and each is scaled
// back up and added on: the larger give a tight halo, the smaller a wide one.
func (l *Lighting) addGlow() {
	sop := &ebiten.DrawRectShaderOptions{}
	sop.Images[0] = l.out
	sop.Uniforms = map[string]any{"From": float32(l.look.GlowFrom)}
	l.bright.Clear()
	l.bright.DrawRectShader(l.w, l.h, l.brightShader, sop)
	src := l.bright
	for _, b := range l.blur {
		b.Clear()
		b.DrawImage(src, resize(src, b, ebiten.BlendSourceOver))
		src = b
	}
	for _, b := range l.blur {
		op := resize(b, l.out, ebiten.BlendLighter)
		k := float32(l.look.Glow)
		op.ColorScale.Scale(k, k, k, k)
		l.out.DrawImage(b, op)
	}
}

// resize is options that draw src over the whole of dst, smoothly.
func resize(src, dst *ebiten.Image, blend ebiten.Blend) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: blend}
	op.GeoM.Scale(float64(dst.Bounds().Dx())/float64(src.Bounds().Dx()),
		float64(dst.Bounds().Dy())/float64(src.Bounds().Dy()))
	return op
}
