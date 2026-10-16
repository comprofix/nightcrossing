package fx

import (
	_ "embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed crt.kage
var crtKage []byte

// The CRT look's strengths (DESIGN §3.3, D10).
const (
	crtScanline = 0.35
	crtMask     = 0.12
	crtVignette = 0.25
	crtBoost    = 1.2
)

// CRT draws the screen as an old monitor would: scanlines, a faint
// aperture grille and darker corners (DESIGN §8: the CRT option). Without
// shader support it draws nothing, and the caller draws the screen plainly.
type CRT struct {
	shader *ebiten.Shader
	tried  bool
}

// Draw draws src, the screen already scaled up by a whole number scale,
// onto dst through geo. It reports false if the shader can't run.
func (c *CRT) Draw(dst, src *ebiten.Image, scale int, geo ebiten.GeoM) bool {
	if !c.tried {
		c.tried = true
		s, err := ebiten.NewShader(crtKage)
		if err != nil {
			log.Printf("CRT shader: %v; drawing without it", err)
		}
		c.shader = s
	}
	if c.shader == nil {
		return false
	}
	op := &ebiten.DrawRectShaderOptions{GeoM: geo}
	op.Images[0] = src
	op.Uniforms = map[string]any{
		"Scale":    float32(scale),
		"Scanline": float32(crtScanline),
		"Mask":     float32(crtMask),
		"Vignette": float32(crtVignette),
		"Boost":    float32(crtBoost),
	}
	b := src.Bounds()
	dst.DrawRectShader(b.Dx(), b.Dy(), c.shader, op)
	return true
}

// ShadersAvailable reports whether all the game's shaders (the lighting,
// glow and CRT) compile here. If not, the game draws without them and the
// options for them are dimmed.
func ShadersAvailable() bool {
	for _, src := range [][]byte{lightKage, brightKage, crtKage} {
		s, err := ebiten.NewShader(src)
		if err != nil {
			log.Printf("shaders: %v; drawing without them", err)
			return false
		}
		s.Deallocate()
	}
	return true
}
