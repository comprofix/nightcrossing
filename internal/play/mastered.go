package play

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/score"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// Mastered is the screen after the bonus tally (DESIGN §6.7, D62): the
// mode is mastered, which opens the next (D64), and the run chooses to
// finish there, with a bonus for each life left, or carry on into the
// loops for score.
type Mastered struct {
	levels config.ModeLevels
	menu   ui.Menu
}

// newMastered is the MASTERED screen for the game in progress, which has
// played levels. Reaching it marks the run and the mode mastered,
// whichever is chosen next; a test game marks no mode (D62–D64).
func newMastered(ctx *scene.Context, levels config.ModeLevels) *Mastered {
	s := ctx.Session
	s.Mastered = true
	if !ctx.Testing() && ctx.Scores.MarkMastered(s.Mode.ID) {
		ctx.ScoresChanged()
	}
	m := &Mastered{levels: levels}
	m.menu.Items = []ui.Item{
		{Label: fmt.Sprintf("Finish: +%d for each life left", config.PointsFinishLife), Confirm: func() { ctx.Switch(m.finish(ctx)) }},
		{Label: "Carry on", Confirm: func() { ctx.Switch(newLevel(ctx, levels, len(levels.Levels)+1)) }},
	}
	m.menu.OnMove = func() { ctx.Audio.PlayEffect(scene.MenuMove) }
	m.menu.OnConfirm = func() { ctx.Audio.PlayEffect(scene.MenuConfirm) }
	return m
}

// Update implements scene.Scene.
func (m *Mastered) Update(ctx *scene.Context) error {
	if ctx.Input.JustPressed(input.Back) {
		return nil // the choice has to be made
	}
	if f := ctx.Fonts; f.Small != nil && f.Large != nil { // headless tests have no fonts
		m.menu.PlaceCentred(f.Small, ctx.Preset.ScreenWidth, masteredMenuTop(ctx))
	}
	m.menu.Update(ctx.Input)
	return nil
}

// masteredTop is the top of the MASTERED screen's first heading, and
// masteredMenuTop of its menu, below the headings and the next mode's line.
func masteredTop(ctx *scene.Context) int {
	return ctx.Preset.ScreenHeight/4 - ctx.Fonts.Large.LineHeight()
}
func masteredMenuTop(ctx *scene.Context) int {
	f := ctx.Fonts
	return masteredTop(ctx) + 2*f.Large.LineHeight() + ui.LineGap + 5*f.Small.LineHeight()
}

// finish adds the life bonus, counted up on its own tally, then goes on
// to the high scores and the next mode's intro, or the MODE menu after the
// last mode (D62).
func (m *Mastered) finish(ctx *scene.Context) scene.Scene {
	s := ctx.Session
	lives := s.Lives
	s.AddScore(lives * config.PointsFinishLife) // no life from these points: the game is over
	line := scene.TallyLine{Label: fmt.Sprintf("LIVES %d x %d", lives, config.PointsFinishLife), Points: lives * config.PointsFinishLife}
	return scene.NewTally("FINISHED", []scene.TallyLine{line}, func() scene.Scene {
		return scene.AfterGame(ctx, s.Mode, entry(s), func() scene.Scene { return afterFinish(ctx) })
	})
}

// afterFinish is where a finished game goes after its high scores: the
// next mode's intro, or the MODE menu after the last (D62).
func afterFinish(ctx *scene.Context) scene.Scene {
	if i, ok := nextMode(ctx); ok {
		return scene.NewIntro(i)
	}
	return scene.NewModeSelect()
}

// nextMode is where in the context's modes the mode after the game in
// progress's is, if there is one.
func nextMode(ctx *scene.Context) (int, bool) {
	modes := ctx.ModeList()
	for i, md := range modes {
		if md.ID == ctx.Session.Mode.ID && i+1 < len(modes) {
			return i + 1, true
		}
	}
	return 0, false
}

// entry is the high score entry for s: its score and marks (D63).
func entry(s *session.Session) score.Entry {
	return score.Entry{Score: s.Score, Mastered: s.Mastered, Continued: s.Continued}
}

// Draw implements scene.Scene.
func (m *Mastered) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	h := ctx.Preset.ScreenHeight
	dst.Fill(gfx.RGBA(sc.Backdrop))
	y := masteredTop(ctx)
	ui.TextCentred(dst, f.Large, "YOU'VE MASTERED", y, sc.Text, sc.Shadow)
	y += f.Large.LineHeight() + ui.LineGap
	ui.TextCentred(dst, f.Large, strings.ToUpper(ctx.Session.Mode.Name), y, sc.Title, sc.Shadow)
	y += f.Large.LineHeight() + 2*f.Small.LineHeight()
	next := "EVERY MODE IS OPEN"
	if i, ok := nextMode(ctx); ok {
		next = strings.ToUpper(ctx.ModeList()[i].Name) + " AWAITS"
	}
	ui.TextCentred(dst, f.Small, next, y, sc.Highlight, sc.Shadow)
	y = masteredMenuTop(ctx)

	mw, mh := m.menu.Size(f.Small)
	ui.Panel(dst, ui.PanelAround(ctx.Preset.ScreenWidth, mw, mh, y, panelPad), sc.Panel, sc.Border)
	m.menu.DrawCentred(dst, f.Small, y, sc)
	ui.TextCentred(dst, f.Small, "Finish banks your lives; carry on risks them for more", h-2*f.Small.LineHeight(), sc.Dim, sc.Shadow)
}

// panelPad is the margin inside the MASTERED screen's panel.
const panelPad = 8
