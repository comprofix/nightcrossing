package scene

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// Intro shows a mode before it is played: what it is, what it brings, what
// carries over from the modes before it, its best score and whether it is
// mastered (D64). Confirm plays it; Back returns to the mode menu.
type Intro struct {
	mode int // index into the context's ModeList
}

// NewIntro returns the intro for the i'th of the context's ModeList.
func NewIntro(i int) *Intro { return &Intro{mode: i} }

// Update implements Scene.
func (s *Intro) Update(ctx *Context) error {
	playTitleTune(ctx)
	switch {
	case ctx.Input.JustPressed(input.Back):
		ctx.Switch(NewModeSelectAt(s.mode))
	case ctx.Input.JustPressed(input.Confirm), ctx.Input.Clicked():
		ctx.Audio.PlayEffect(MenuConfirm)
		m := ctx.ModeList()[s.mode]
		ctx.Switch(ctx.makeGame(func(c *Context) Scene { return c.NewPlay(c, m) }))
	}
	return nil
}

// Draw implements Scene.
func (s *Intro) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	m := ctx.ModeList()[s.mode]
	dst.Fill(gfx.RGBA(sc.Backdrop))
	line := f.Small.LineHeight() + ui.LineGap

	y := f.Large.LineHeight()
	ui.TextCentred(dst, f.Large, strings.ToUpper(m.Name), y, sc.Title, sc.Shadow)
	y += f.Large.LineHeight() + line
	ui.TextCentred(dst, f.Small, m.Desc, y, sc.Text, sc.Shadow)
	y += 2 * line

	// The lists line up in two columns, the block centred on the screen.
	nameW := 0
	for _, n := range m.New {
		nameW = max(nameW, f.Small.Width(n.Name))
	}
	descX := 3*f.Small.CellWidth() + nameW + 2*f.Small.CellWidth()
	w := f.Small.Width("NEW IN THIS MODE")
	for _, n := range m.New {
		w = max(w, descX+f.Small.Width(n.Desc))
	}
	for _, a := range m.Also {
		w = max(w, 2*f.Small.CellWidth()+f.Small.Width(a))
	}
	x := (ctx.Preset.ScreenWidth - w) / 2

	f.Small.DrawShadowed(dst, "NEW IN THIS MODE", x, y, sc.Title, sc.Shadow)
	y += line
	for _, n := range m.New {
		f.Small.DrawShadowed(dst, ">", x+f.Small.CellWidth(), y, sc.Dim, sc.Shadow)
		f.Small.DrawShadowed(dst, n.Name, x+3*f.Small.CellWidth(), y, sc.Highlight, sc.Shadow)
		f.Small.DrawShadowed(dst, n.Desc, x+descX, y, sc.Text, sc.Shadow)
		y += line
	}
	if len(m.Also) > 0 {
		y += line
		f.Small.DrawShadowed(dst, "ALSO", x, y, sc.Title, sc.Shadow)
		y += line
		for _, a := range m.Also {
			f.Small.DrawShadowed(dst, a, x+2*f.Small.CellWidth(), y, sc.Text, sc.Shadow)
			y += line
		}
	}

	y += line
	best := 0
	if t := ctx.Scores.Table(m.ID); len(t) > 0 {
		best = t[0].Score
	}
	f.Small.DrawShadowed(dst, fmt.Sprintf("BEST  %06d", best), x, y, sc.Text, sc.Shadow)
	if ctx.Scores.IsMastered(m.ID) {
		c := Tick + " MASTERED" // D64
		f.Small.DrawShadowed(dst, c, x+w-f.Small.Width(c), y, sc.Highlight, sc.Shadow)
	}

	ui.TextCentred(dst, f.Small, "ENTER: PLAY   ESC: BACK", ctx.Preset.ScreenHeight-2*f.Small.LineHeight(), sc.Dim, sc.Shadow)
}
