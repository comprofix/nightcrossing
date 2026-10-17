// Package play is Night Crossing's play scene. It runs the world one tick at
// a time, draws its state with the status bar and hint strip, and turns the
// world's events into sound and effects. The rules live in package world.
package play

import (
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/ui"
	"github.com/comprofix/nightcrossing/internal/world"
)

// Sheet is the name of the game's sprite sheet.
const Sheet = "nightcrossing"

// Status bar layout, in screen pixels from the left (DESIGN §3.2).
const (
	hudHiX     = 136
	hudLevelX  = 240
	hudLivesX  = 304
	hudLivesW  = 100 // room for the lives before the combo and the timer
	hudLifeGap = 2
	timeBarW   = 120
	timeBarH   = 8
	labelGap   = 8 // between a label and what it labels
)

// hint is the help shown in the strip below the playfield.
const hint = "ARROWS/WASD: HOP   ESC: PAUSE   CTRL+R: RESTART"

// Effects and messages, in ticks and pixels.
const (
	shakeTicks     = 12
	shakePx        = 3
	softShakePx    = 2                     // a death in the water, to a heron or the clock (DESIGN §3.3)
	gameOverTicks  = 2 * config.TPS        // the game over message shows this long
	bannerTicks    = 2 * config.TPS        // the level number shows this long at the start
	levelHintTicks = config.LevelCalmTicks // a new level's hint shows this long, before any event (DESIGN §6.1)
	clearTicks     = config.TPS            // the cleared level shows this long before the tally
)

// Play is a level of a game in progress.
type Play struct {
	world  *world.World
	hero   Hero
	levels config.ModeLevels
	ticks  int // since the level started
	clear  int // ticks since the level was cleared; 0 until then

	light     *fx.Lighting
	parts     *fx.Particles
	nookGlow  [len(config.Nooks)]int // ticks left of each nook's flare as a toad arrives
	outline   *ebiten.Image          // a drawing buffer for high contrast's outlines
	pops      []pop                  // points popping up over the toad (D38)
	sky       int                    // where the day and night cycle is (D39; see sky.go)
	overcast  int                    // how far fog or rain has dimmed the light, in ticks (D44)
	fogIn     int                    // how far the fog is in, in ticks
	rainIn    int                    // how far the rain is in, in ticks
	windIn    int                    // how far a dry storm's wind is in, in ticks (D47)
	windDir   int                    // the way it blows, kept as it dies away
	stormSky  int                    // how far a storm's sky has rolled in, in ticks (D49, D51)
	flash     int                    // ticks since a storm's last lightning flash (D49)
	flashes   int                    // how many flashes this level, for the next one's timing
	nextFlash int                    // ticks until the next flash, in a storm
	storming  bool                   // a storm was on last tick, so lightning knows when one begins
	fogImg    *ebiten.Image          // a drawing buffer: one fog patch
}

// New starts a game in a mode, on its first level, or on ctx.StartLevel
// in a debug build started with -level.
func New(ctx *scene.Context, mode config.Mode) scene.Scene {
	ctx.Session = session.New(mode, config.StartLives)
	return newLevel(ctx, mode.Levels, max(ctx.StartLevel, 1))
}

// newLevel starts level n of levels for the game in progress.
func newLevel(ctx *scene.Context, levels config.ModeLevels, n int) *Play {
	return newPlay(ctx, levels, levels.Level(n))
}

// newPlay starts a given level for the game in progress, so tests can use
// levels of their own.
func newPlay(ctx *scene.Context, levels config.ModeLevels, level config.Level) *Play {
	level.Storms = ctx.Storms // -storm, for testing (D52)
	return build(ctx.Session, levels, level, ctx.RNG, !ctx.NoShaders)
}

// build is a level of the game in s, its random choices from rng, lit by
// shaders if shaders is true.
func build(s *session.Session, levels config.ModeLevels, level config.Level, rng *rand.Rand, shaders bool) *Play {
	w := world.New(s, level, rng)
	pr := config.Project
	p := &Play{
		world:  w,
		hero:   Toad,
		levels: levels,
		light:  fx.NewLighting(pr.ScreenWidth, pr.ScreenHeight, shaders),
		parts:  fx.NewParticles(rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))),
		sky:    skyBefore,
		flash:  noFlash,
	}
	return p
}

// World is the game's state, for tests.
func (p *Play) World() *world.World { return p.world }

// Update implements scene.Scene.
func (p *Play) Update(ctx *scene.Context) error {
	in := ctx.Input
	switch {
	case in.JustPressed(input.Pause):
		ctx.Cut(scene.NewPause(p))
		return nil
	case in.JustPressed(input.Menu):
		ctx.Switch(scene.NewMainMenu())
		return nil
	case in.JustPressed(input.Restart):
		ctx.Switch(New(ctx, ctx.Session.Mode))
		return nil
	}
	p.ticks++
	p.updateEffects()
	if p.thunderNow() {
		ctx.Audio.PlayEffect(DistantThunder) // a beat after the flash (D49)
	}
	if p.clear > 0 {
		if p.clear++; p.clear > clearTicks {
			ctx.Switch(p.tally(ctx))
		}
		return nil
	}
	if ctx.Audio.Tune() != Stroll {
		ctx.Audio.PlayTune(Stroll) // the gameplay loop (DESIGN §3.4)
	}
	for _, e := range p.world.Update(pressed(in)) {
		p.effects(e)
		for _, s := range p.soundsFor(e) {
			ctx.Audio.PlayEffect(s)
		}
		switch e {
		case world.Squashed, world.Bumped:
			ctx.Shake.Start(shakeTicks, shakePx)
		case world.Splashed, world.Swept, world.Caught, world.TimeUp:
			ctx.Shake.Start(shakeTicks, softShakePx)
		case world.Struck:
			ctx.Shake.Start(shakeTicks, struckShakePx)
		case world.TrainComing:
			ctx.Shake.Start(trainShakeTicks, trainShakePx) // DESIGN §3.3
		case world.LevelClear:
			p.clear = 1
			ctx.Audio.PlayTune(LevelClearJingle)
			p.markCleared(ctx)
		case world.GameOver:
			ctx.Audio.PlayTune(GameOverJingle)
			ctx.Cut(newEnded(p))
		}
	}
	p.popGained()
	if e := p.railEffect(); e != nil {
		ctx.Audio.PlayEffect(e)
	}
	return nil
}

// tally is the level-clear tally of what this level earned, then the next
// level (DESIGN §7). The points are already in the score; the tally shows
// where they came from.
func (p *Play) tally(ctx *scene.Context) scene.Scene {
	st := p.world.Stats
	n := p.world.Level.Number
	lines := []scene.TallyLine{
		{Label: "HOPS", Points: st.Hops},
		{Label: fmt.Sprintf("HOME %d x %d", st.Goals, config.PointsGoal), Points: st.Goals * config.PointsGoal},
		{Label: "TIME BONUS", Points: st.TimeBonus},
		{Label: "ALL FIVE HOME", Points: st.Clear},
	}
	if p.world.Level.Toadlets > 0 {
		lines = append(lines, scene.TallyLine{Label: fmt.Sprintf("TOADLETS %d x %d", st.Toadlets, config.PointsToadlet),
			Points: st.Toadlets * config.PointsToadlet}) // D53
	}
	if p.world.Level.Combo {
		lines = append(lines, scene.TallyLine{Label: "COMBO BONUS", Points: st.Combo})
	}
	return scene.NewTally(fmt.Sprintf("LEVEL %d CLEAR", n), lines, func() scene.Scene {
		return newLevel(ctx, p.levels, n+1)
	})
}

// markCleared records the mode as cleared once its last level, level 8,
// is cleared, for the MODE menu (D24).
func (p *Play) markCleared(ctx *scene.Context) {
	if ctx.Testing() {
		return // a test game, started part way or with one kind of storm
	}
	if p.world.Level.Number >= len(p.levels.Levels) && ctx.Scores.MarkCleared(ctx.Session.Mode.ID) {
		ctx.ScoresChanged()
	}
}

// Ended holds the last moment of the game on screen under a GAME OVER
// message, then goes on to the high scores. Confirm moves on sooner.
type Ended struct {
	play  *Play
	ticks int
}

func newEnded(p *Play) *Ended { return &Ended{play: p} }

// Update implements scene.Scene.
func (e *Ended) Update(ctx *scene.Context) error {
	e.ticks++
	if e.ticks >= gameOverTicks || (e.ticks > config.TPS/2 && ctx.Input.JustPressed(input.Confirm)) {
		s := ctx.Session
		ctx.Switch(scene.AfterGame(ctx, s.Mode, s.Score))
	}
	return nil
}

// Draw implements scene.Scene.
func (e *Ended) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	e.play.Draw(ctx, dst)
	scene.Dim(dst, sc.Shadow, 0.5)
	ui.TextCentred(dst, f.Large, "GAME OVER", (config.Project.ScreenHeight-f.Large.Size)/2, sc.Title, sc.Shadow)
}

// pressed is the direction pressed this tick: the first of up, down, left
// and right. Each press is one hop; holding a key does not repeat it
// (DESIGN §5.1).
func pressed(in *input.Input) world.Dir {
	switch {
	case in.JustPressed(input.Up):
		return world.Up
	case in.JustPressed(input.Down):
		return world.Down
	case in.JustPressed(input.Left):
		return world.Left
	case in.JustPressed(input.Right):
		return world.Right
	}
	return world.None
}

// Draw implements scene.Scene.
func (p *Play) Draw(ctx *scene.Context, dst *ebiten.Image) {
	dst.Fill(gfx.RGBA(ctx.Scheme.Backdrop))
	p.drawField(ctx, dst)
	p.drawTrafficLights(ctx, dst)
	p.drawPops(ctx, dst)
	p.drawHUD(ctx, dst)
	p.drawHint(ctx, dst)
	p.drawBanner(ctx, dst)
}

// drawHUD draws the status bar: score, best score, level, lives and time.
func (p *Play) drawHUD(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	s := ctx.Session
	ty := (config.HUDHeight - f.Small.Size) / 2

	f.Small.DrawShadowed(dst, fmt.Sprintf("SCORE %06d", s.Score), config.FieldMargin, ty, sc.Text, sc.Shadow)
	f.Small.DrawShadowed(dst, fmt.Sprintf("HI %06d", max(best(ctx), s.Score)), hudHiX, ty, sc.Text, sc.Shadow)
	f.Small.DrawShadowed(dst, fmt.Sprintf("LV %d", p.world.Level.Number), hudLevelX, ty, sc.Text, sc.Shadow)

	life := ctx.Sheet(Sheet).Sprite("life")
	ly := float64((config.HUDHeight - life.H) / 2)
	for i := range min(s.Lives, hudLivesW/(life.W+hudLifeGap)) {
		life.Draw(dst, 0, float64(hudLivesX+i*(life.W+hudLifeGap)), ly, false)
	}

	sw := config.Project.ScreenWidth
	barX := sw - config.FieldMargin - timeBarW
	barY := (config.HUDHeight - timeBarH) / 2
	label := "TIME"
	f.Small.DrawShadowed(dst, label, barX-labelGap-len(label)*f.Small.Size, ty, sc.Text, sc.Shadow)
	w := p.world
	filled := timeBarW * max(w.TicksLeft, 0) / w.Level.TimerTicks
	bar := resurrect64.Emerald
	if w.TicksLeft <= config.LowTimeSecs*config.TPS {
		bar = resurrect64.Red // time is low (DESIGN §5.4)
	}
	vector.FillRect(dst, float32(barX), float32(barY), timeBarW, timeBarH, gfx.RGBA(sc.Panel), false)
	vector.FillRect(dst, float32(barX), float32(barY), float32(filled), timeBarH, gfx.RGBA(bar), false)
	p.drawCombo(ctx, dst, ty)
}

// drawHint draws the strip below the playfield: the controls, or for a
// while at the start of a level that brings in something new, its hint
// (DESIGN §7).
func (p *Play) drawHint(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	top := config.FieldY + config.Rows*config.Cell
	text, c := hint, sc.Dim
	if h := p.world.Level.Hint; h != "" && p.ticks < levelHintTicks {
		text, c = h, sc.Highlight
	}
	if m := p.eventMessage(); m != "" {
		text, c = m, sc.Highlight // an event is announced (DESIGN §6.1)
	}
	ui.TextCentred(dst, f.Small, text, top+(config.HintHeight-f.Small.Size)/2, c, sc.Shadow)
}

// drawBanner shows the level number as the level starts, and LEVEL CLEAR
// once it is cleared, on the verge between the river and the road.
func (p *Play) drawBanner(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	var text string
	switch {
	case p.clear > 0:
		text = "LEVEL CLEAR"
	case p.ticks < bannerTicks:
		text = fmt.Sprintf("LEVEL %d", p.world.Level.Number)
	default:
		return
	}
	const vergeRow = 6
	y := config.FieldY + vergeRow*config.Cell + (config.Cell-f.Large.Size)/2
	ui.TextCentred(dst, f.Large, text, y, sc.Title, sc.Shadow)
}

// best is the top score in the current mode's table, or 0.
func best(ctx *scene.Context) int {
	if es := ctx.Scores.Table(ctx.Session.Mode.ID); len(es) > 0 {
		return es[0].Score
	}
	return 0
}
