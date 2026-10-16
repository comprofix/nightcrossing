package play

import "github.com/comprofix/nightcrossing/internal/audio"

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
)

// Toad is the hero for 1.0 (DESIGN §2).
var Toad = Hero{Name: "Toad", Sprite: "toad", Splash: "splash", Hop: ToadHop}
