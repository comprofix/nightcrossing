// Package scene runs the game's screens: the scene manager with fades and
// screen shake, and the shared scenes.
package scene

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/audio"
	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
	"github.com/comprofix/gameskeleton/internal/score"
	"github.com/comprofix/gameskeleton/internal/session"
)

// Scene is one screen of the game.
type Scene interface {
	// Update advances the scene by one tick.
	Update(ctx *Context) error
	// Draw renders onto the internal screen. It must not change state.
	Draw(ctx *Context, dst *ebiten.Image)
}

// NewRNG is the random number generator for a seed. The game and tests
// both use it, so a seed gives the same game in each.
func NewRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

// Context is the shared state handed to scenes. Games add their own fields.
type Context struct {
	Input    *input.Input
	RNG      *rand.Rand
	Fonts    gfx.Fonts
	Scheme   gfx.Scheme
	Sheets   map[string]*gfx.Sheet
	Settings *config.Settings
	Shake    Shaker
	Tick     int64            // ticks since the game started
	Audio    *audio.Synth     // nil in tests: every call is then ignored
	Sounds   audio.Bank       // every tune and effect, for the sound test
	Session  *session.Session // the game in progress, if any
	Scores   *score.Tables

	// NewPlay starts a game at the given skill. The game package sets it,
	// so the shared scenes don't depend on the game's own packages.
	NewPlay func(ctx *Context, skill config.Skill) Scene

	next            Scene
	cut             bool
	quit            bool
	settingsChanged bool
	scoresChanged   bool
}

// Switch asks the manager to fade to s at the end of this tick.
func (c *Context) Switch(s Scene) { c.next, c.cut = s, false }

// Cut asks the manager to change to s at the end of this tick without a
// fade, e.g. to open or close the pause menu.
func (c *Context) Cut(s Scene) { c.next, c.cut = s, true }

// Quit asks the game to save and close.
func (c *Context) Quit() { c.quit = true }

// Quitting reports whether a scene asked to quit.
func (c *Context) Quitting() bool { return c.quit }

// SettingsChanged records that a scene changed the settings, so the game
// applies and saves them.
func (c *Context) SettingsChanged() { c.settingsChanged = true }

// TakeSettingsChanged reports whether the settings changed since the last
// call, and clears the flag.
func (c *Context) TakeSettingsChanged() bool {
	ch := c.settingsChanged
	c.settingsChanged = false
	return ch
}

// ScoresChanged records that a score was added, so the game saves the
// tables.
func (c *Context) ScoresChanged() { c.scoresChanged = true }

// TakeScoresChanged reports whether the scores changed since the last call,
// and clears the flag.
func (c *Context) TakeScoresChanged() bool {
	ch := c.scoresChanged
	c.scoresChanged = false
	return ch
}

// Sheet returns the named sprite sheet. It panics if it is missing, since
// sheets are fixed at build time.
func (c *Context) Sheet(name string) *gfx.Sheet {
	sh, ok := c.Sheets[name]
	if !ok {
		panic("scene: no sprite sheet named " + name)
	}
	return sh
}
