// Package play is Night Crossing's play scene. It runs the world one tick at
// a time, draws its state with the status bar and hint strip, and turns the
// world's events into sound and effects. The rules live in package world.
package play

import (
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/audio"
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
	hudLivesX  = 296 // room for config.MaxLives lives before the combo (D57)
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
	cryLeadTicks   = 10                    // a toadlet's cry comes this far ahead of its death's sound (D56)
	peepGapTicks   = 8                     // the toadlets getting home peep this far apart, after the home run (D56)
)

// Play is a level of a game in progress.
type Play struct {
	world  *world.World
	hero   Hero
	levels config.ModeLevels
	ticks  int         // since the level started
	clear  int         // ticks since the level was cleared; 0 until then
	ending world.Event // how it ended: LevelClear, or on a bonus level AllCaught or BonusOver (D61)
	later  []lateSound // sounds waiting to play
	// sirenCut: the emergency vehicle has run the hero down, and its siren
	// is cut off until it has gone (D71).
	sirenCut bool

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
// in a debug build started with -level, or on its bonus level with
// -bonus (D61).
func New(ctx *scene.Context, mode config.Mode) scene.Scene {
	ctx.Session = session.New(mode, config.StartLives)
	if lv, ok := config.BonusLevel(mode); ctx.Bonus && ok {
		return newPlay(ctx, mode.Levels, lv)
	}
	return newLevel(ctx, mode.Levels, max(ctx.StartLevel, 1))
}

// newLevel starts level n of levels for the game in progress. Once the run
// has mastered its mode, each level it reaches is its CONTINUED mark
// (D63).
func newLevel(ctx *scene.Context, levels config.ModeLevels, n int) *Play {
	if s := ctx.Session; s.Mastered {
		s.Continued = n
	}
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
		sky:    skyStart(level),
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
		if p.clear++; p.clear > p.clearFor() {
			ctx.Switch(p.tally(ctx))
		}
		return nil
	}
	tune := Stroll // the gameplay loop (DESIGN §3.4)
	if p.world.Bonus() {
		tune = BonusNight // D61
	}
	if ctx.Audio.Tune() != tune {
		ctx.Audio.PlayTune(tune)
	}
	p.playLater(ctx)
	for _, e := range p.world.Update(pressed(in)) {
		p.effects(e)
		p.sound(ctx, e)
		switch e {
		case world.Squashed, world.Bumped:
			ctx.Shake.Start(shakeTicks, shakePx)
		case world.RunDown:
			ctx.Shake.Start(shakeTicks, shakePx)
			p.sirenCut = true
		case world.Splashed, world.Swept, world.Caught, world.Snatched, world.Nipped, world.Bitten, world.TimeUp, world.Slipped:
			ctx.Shake.Start(shakeTicks, softShakePx)
		case world.Struck:
			ctx.Shake.Start(shakeTicks, struckShakePx)
		case world.ExtraLife:
			p.popUp("1UP", oneUpLiftPx) // from points or a firefly (D57)
		case world.TrainComing:
			ctx.Shake.Start(trainShakeTicks, trainShakePx) // DESIGN §3.3
		case world.LevelClear, world.AllCaught, world.BonusOver:
			p.clear, p.ending = 1, e
			ctx.Audio.PlayTune(LevelClearJingle)
			if p.departing() {
				ctx.Audio.PlayEffect(Horn) // the last train pulls out (DESIGN §6.7)
			}
		case world.GameOver:
			ctx.Audio.PlayTune(GameOverJingle)
			ctx.Cut(newEnded(p))
		}
	}
	p.popGained()
	if e := p.railEffect(); e != nil {
		ctx.Audio.PlayEffect(e)
	}
	if p.world.Siren.Phase == world.SirenAway {
		p.sirenCut = false
	}
	if e := p.sirenEffect(); e != nil {
		ctx.Audio.PlayEffect(e)
	}
	return nil
}

// tally is the level-clear tally of what this level earned, then what
// comes next (DESIGN §7, §6.7): the next level; after level 8, the mode's
// bonus level, or the MASTERED screen for a mode with none yet; and after
// the bonus level, the MASTERED screen (D61, D62). A test game started
// part way goes straight on to level 9. The points are already in the
// score; the tally shows where they came from.
func (p *Play) tally(ctx *scene.Context) scene.Scene {
	w := p.world
	n := w.Level.Number
	if b := w.Level.Bonus; b != nil {
		mastered := func() scene.Scene { return newMastered(ctx, p.levels) }
		if b.Ending { // the school gates: the game's ending and credits first (D62)
			return scene.NewTally(b.Name, p.bonusLines(), func() scene.Scene { return newEnding(w.Stats.All > 0, mastered) })
		}
		return scene.NewTally(b.Name, p.bonusLines(), mastered)
	}
	next := func() scene.Scene { return newLevel(ctx, p.levels, n+1) }
	if n == len(p.levels.Levels) && !ctx.Testing() {
		next = func() scene.Scene {
			if lv, ok := config.BonusLevel(ctx.Session.Mode); ok {
				return newPlay(ctx, p.levels, lv)
			}
			return newMastered(ctx, p.levels)
		}
	}
	return scene.NewTally(fmt.Sprintf("LEVEL %d CLEAR", n), p.tallyLines(), next)
}

// bonusLines are the bonus level's tally: the fireflies or the fruit,
// collecting the whole set, the time bonus for doing it early, and any
// nooks reached (D61, DESIGN §6.7).
func (p *Play) bonusLines() []scene.TallyLine {
	st, b := p.world.Stats, p.world.Level.Bonus
	var lines []scene.TallyLine
	if b.Fireflies > 0 {
		name := "FIREFLIES"
		if b.Dawn {
			name = "DRAGONFLIES"
		}
		lines = append(lines, scene.TallyLine{Label: fmt.Sprintf("%s %d x %d", name, st.Fireflies, b.Firefly),
			Points: st.Fireflies * b.Firefly})
	}
	if b.RunMax > 0 {
		lines = append(lines, scene.TallyLine{Label: fmt.Sprintf("BEST RUN %d", st.BestRun), Points: st.Run})
	}
	if b.Stragglers > 0 {
		lines = append(lines,
			scene.TallyLine{Label: fmt.Sprintf("TOADLETS HOME %d x %d", st.Toadlets, b.Straggler), Points: st.Toadlets * b.Straggler},
			scene.TallyLine{Label: "BIG TRIPS", Points: st.Trip})
	}
	if len(b.Fruit) > 0 {
		lines = append(lines, scene.TallyLine{Label: fmt.Sprintf("FRUIT %d/%d", st.Picked, len(b.Fruit)), Points: st.Fruit})
	}
	lines = append(lines,
		scene.TallyLine{Label: p.allLabel(), Points: st.All},
		scene.TallyLine{Label: "TIME BONUS", Points: st.TimeBonus})
	if st.Goals > 0 {
		lines = append(lines, scene.TallyLine{Label: fmt.Sprintf("NOOKS %d x %d", st.Goals, config.PointsBonusNook),
			Points: st.Goals * config.PointsBonusNook})
	}
	return lines
}

// tallyLines are the tally's lines: one for each kind of points the level
// can earn.
func (p *Play) tallyLines() []scene.TallyLine {
	st := p.world.Stats
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
	if p.world.Level.FruitPts > 0 {
		lines = append(lines, scene.TallyLine{Label: "FRUIT", Points: st.Fruit}) // D59
	}
	if st.AtCap > 0 {
		lines = append(lines, scene.TallyLine{Label: "FIREFLY BONUS", Points: st.AtCap}) // fireflies at full lives (D57)
	}
	if p.world.Level.Combo {
		lines = append(lines, scene.TallyLine{Label: "COMBO BONUS", Points: st.Combo})
	}
	return lines
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
		ctx.Switch(scene.AfterGame(ctx, s.Mode, entry(s), nil))
	}
	return nil
}

// Draw implements scene.Scene.
func (e *Ended) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	e.play.Draw(ctx, dst)
	scene.Dim(dst, sc.Shadow, 0.5)
	ui.TextCentred(dst, f.Large, "GAME OVER", (config.Project.ScreenHeight-f.Large.Size())/2, sc.Title, sc.Shadow)
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
	ty := (config.HUDHeight - f.Small.Size()) / 2

	f.Small.DrawShadowed(dst, fmt.Sprintf("SCORE %06d", s.Score), config.FieldMargin, ty, sc.Text, sc.Shadow)
	f.Small.DrawShadowed(dst, fmt.Sprintf("HI %06d", max(best(ctx), s.Score)), hudHiX, ty, sc.Text, sc.Shadow)
	level := fmt.Sprintf("LV %d", p.world.Level.Number)
	if p.world.Bonus() {
		level = "BONUS"
	}
	f.Small.DrawShadowed(dst, level, hudLevelX, ty, sc.Text, sc.Shadow)

	life := ctx.Sheet(Sheet).Sprite("life")
	ly := float64((config.HUDHeight - life.H) / 2)
	for i := range min(s.Lives, config.MaxLives) {
		life.Draw(dst, 0, float64(hudLivesX+i*(life.W+hudLifeGap)), ly, false)
	}

	sw := config.Project.ScreenWidth
	barX := sw - config.FieldMargin - timeBarW
	barY := (config.HUDHeight - timeBarH) / 2
	label := "TIME"
	f.Small.DrawShadowed(dst, label, barX-labelGap-len(label)*f.Small.Size(), ty, sc.Text, sc.Shadow)
	w := p.world
	filled := timeBarW * max(w.TicksLeft, 0) / w.Level.TimerTicks
	bar := resurrect64.Emerald
	if w.TicksLeft <= config.LowTimeSecs*config.TPS {
		bar = resurrect64.Red // time is low (DESIGN §5.4)
	}
	vector.FillRect(dst, float32(barX), float32(barY), timeBarW, timeBarH, gfx.RGBA(sc.Panel), false)
	vector.FillRect(dst, float32(barX), float32(barY), float32(filled), timeBarH, gfx.RGBA(bar), false)
	p.drawCombo(ctx, dst, ty)
	p.drawCaught(ctx, dst, ty)
}

// drawCaught draws, on a bonus level, how much of its set the toad has
// collected, with a firefly or a fruit, in the combo's place: there are
// no hop points to combo (D61, DESIGN §6.7).
func (p *Play) drawCaught(ctx *scene.Context, dst *ebiten.Image, ty int) {
	b := p.world.Level.Bonus
	if b == nil {
		return
	}
	sc, f := ctx.Scheme, ctx.Fonts
	sp, frame := ctx.Sheet(Sheet).Sprite(p.hero.Pickup), 0
	switch {
	case b.Stragglers > 0:
		sp = ctx.Sheet(Sheet).Sprite(p.young())
	case b.Fireflies == 0:
		sp = ctx.Sheet(Sheet).Sprite("fruit")
		frame = sp.Frames() - 1 // the watermelon, the best of them
	case b.Dawn:
		sp = ctx.Sheet(Sheet).Sprite(p.hero.DayPickup)
	}
	sp.Draw(dst, frame, hudComboX, float64((config.HUDHeight-sp.H)/2), false)
	count := fmt.Sprintf("%d/%d", p.world.Collected(), b.Size())
	x := hudComboX + sp.W + hudLifeGap
	f.Small.DrawShadowed(dst, count, x, ty, sc.Highlight, sc.Shadow)
	if m := p.world.RunMult(); m > 1 { // what the next catch in the run is worth, before the icon (DESIGN §6.7)
		label := fmt.Sprintf("x%d", m)
		f.Small.DrawShadowed(dst, label, hudComboX-(len(label)+1)*f.Small.Size(), ty, sc.Title, sc.Shadow)
	}
}

// allLabel is what collecting a bonus level's whole set is called: ALL
// CAUGHT for fireflies, ALL COLLECTED for fruit, ALL HOME for toadlets.
func (p *Play) allLabel() string {
	switch b := p.world.Level.Bonus; {
	case b != nil && b.Stragglers > 0:
		return "ALL HOME"
	case b != nil && b.Fireflies == 0:
		return "ALL COLLECTED"
	}
	return "ALL CAUGHT"
}

// overLabel is what a bonus level ending short of the whole set is
// called: TIME UP as the clock runs out, ALL GONE with time still on it,
// when nothing is left to collect (DESIGN §6.7).
func (p *Play) overLabel() string {
	if p.world.TicksLeft > 0 {
		return "ALL GONE"
	}
	return "TIME UP"
}

// levelHints are the hints of the level, for what it brings in: its own,
// and a predator coming in (D66).
func (p *Play) levelHints() []string {
	var hs []string
	for _, h := range []string{p.world.Level.Hint, p.world.Level.PreyHint} {
		if h != "" {
			hs = append(hs, h)
		}
	}
	return hs
}

// drawHint draws the strip below the playfield: the controls, or for a
// while at the start of a level that brings in something new, its hint
// (DESIGN §7).
func (p *Play) drawHint(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	top := config.FieldY + config.Rows*config.Cell
	text, c := hint, sc.Dim
	if hs := p.levelHints(); len(hs) > 0 && p.ticks < levelHintTicks {
		text, c = hs[p.ticks*len(hs)/levelHintTicks], sc.Highlight // in turn, sharing the time (D66)
	}
	if m := p.eventMessage(); m != "" {
		text, c = m, sc.Highlight // an event is announced (DESIGN §6.1)
	}
	ui.TextCentred(dst, f.Small, text, top+(config.HintHeight-f.Small.Size())/2, c, sc.Shadow)
}

// drawBanner shows the level number as the level starts, and LEVEL CLEAR
// once it is cleared, on the verge between the river and the road.
func (p *Play) drawBanner(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	var text string
	switch {
	case p.clear > 0:
		text = map[world.Event]string{world.AllCaught: p.allLabel(), world.BonusOver: p.overLabel()}[p.ending]
		if text == "" {
			text = "LEVEL CLEAR"
		}
	case p.ticks < bannerTicks:
		text = fmt.Sprintf("LEVEL %d", p.world.Level.Number)
		if b := p.world.Level.Bonus; b != nil {
			text = b.Name // the title card (D61)
		}
	default:
		return
	}
	const vergeRow = 6
	y := config.FieldY + vergeRow*config.Cell + (config.Cell-f.Large.Size())/2
	ui.TextCentred(dst, f.Large, text, y, sc.Title, sc.Shadow)
}

// best is the top score in the current mode's table, or 0.
func best(ctx *scene.Context) int {
	if es := ctx.Scores.Table(ctx.Session.Mode.ID); len(es) > 0 {
		return es[0].Score
	}
	return 0
}

// lateSound is an effect waiting to play at a tick of the level.
type lateSound struct {
	at     int
	effect *audio.Effect
}

// sound plays event e's effects. A toadlet's death cries first, with the
// death's own sound a beat later, and a nook filled with toadlets peeps for
// each of them after the home run (D56).
func (p *Play) sound(ctx *scene.Context, e world.Event) {
	delay := 0
	if f := p.world.Fallen(); f != nil && f != &p.world.Player && e == f.Death && p.world.Dying == config.DeathTicks {
		ctx.Audio.PlayEffect(p.hero.Cry)
		delay = cryLeadTicks
	}
	for _, s := range p.soundsFor(e) {
		p.playIn(ctx, s, delay)
	}
	if e == world.Goal {
		run := int(audio.Length(Home.Part.Notes) / audio.Ticks(1))
		for i := range p.world.Line {
			p.playIn(ctx, p.hero.Peep, run+i*peepGapTicks)
		}
	}
}

// playIn plays s in ticks ticks, or now for 0.
func (p *Play) playIn(ctx *scene.Context, s *audio.Effect, ticks int) {
	if ticks == 0 {
		ctx.Audio.PlayEffect(s)
		return
	}
	p.later = append(p.later, lateSound{at: p.ticks + ticks, effect: s})
}

// playLater plays the waiting effects that are due.
func (p *Play) playLater(ctx *scene.Context) {
	kept := p.later[:0]
	for _, s := range p.later {
		if s.at <= p.ticks {
			ctx.Audio.PlayEffect(s.effect)
		} else {
			kept = append(kept, s)
		}
	}
	p.later = kept
}
