package play

import (
	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
)

// Hero is how a hero looks and sounds. Every hero uses the same rules, so
// it is data only and adding one needs no change to the world (DESIGN §2.1).
type Hero struct {
	Name   string
	Sprite string        // facing up, with the frames below
	Splash string        // falling in the water, played through over the death
	Hop    *audio.Effect // played as each hop starts
}

// A hero sprite's frames.
const (
	frameRest     = 0
	frameLeap     = 1 // in the air
	frameLand     = 2 // the landing squash
	frameSquashed = 3 // flattened by a vehicle
	frameCrouch   = 4 // gathering for a hop
	frameStretch  = 5 // pushing off
	frameTuck     = 6 // legs coming forward to land
)

// hopFrames is the hero's frame at each tick of a hop, from its first
// (DESIGN §3.3: a 6–8 frame arc, then the landing squash).
var hopFrames = [config.HopTicks]int{
	frameCrouch, frameStretch, frameStretch, frameLeap,
	frameLeap, frameTuck, frameTuck, frameTuck,
}

// Toad is the hero for 1.0 (DESIGN §2).
var Toad = Hero{Name: "Toad", Sprite: "toad", Splash: "splash", Hop: ToadHop}
