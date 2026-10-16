package scene

import (
	"math/rand/v2"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/score"
)

// harness runs scenes headless: Update only, never Draw.
type harness struct {
	t   *testing.T
	ctx *Context
	mgr *Manager
	in  *input.Script
}

func newHarness(t *testing.T, first Scene) *harness {
	set := config.DefaultSettings(config.Project)
	script := &input.Script{}
	h := &harness{
		t:   t,
		in:  script,
		mgr: NewManager(first),
		ctx: &Context{
			Input:    input.New(script),
			RNG:      rand.New(rand.NewPCG(1, 2)),
			Settings: &set,
			NewPlay:  func(*Context, config.Skill) Scene { return &stubPlay{} },
			Scores:   score.New(),
		},
	}
	h.ticks(FadeTicks) // the opening fade-in
	return h
}

// ticks runs n ticks.
func (h *harness) ticks(n int) {
	for range n {
		h.ctx.Input.Update()
		if err := h.mgr.Update(h.ctx); err != nil {
			h.t.Fatal(err)
		}
		h.ctx.Tick++
	}
}

// press presses keys for one tick, releases them, then waits out any fade.
func (h *harness) press(keys ...ebiten.Key) {
	h.in.Steps = append(h.in.Steps, input.Press(keys...), input.Idle(1))
	h.ticks(2)
	h.ticks(2 * FadeTicks)
}

// stubPlay stands in for a game: it pauses on Pause and counts updates.
type stubPlay struct{ updates int }

func (p *stubPlay) Update(ctx *Context) error {
	p.updates++
	if ctx.Input.JustPressed(input.Pause) {
		ctx.Cut(NewPause(p))
	}
	return nil
}
func (p *stubPlay) Draw(*Context, *ebiten.Image) {}

func current[T Scene](t *testing.T, h *harness) T {
	t.Helper()
	s, ok := h.mgr.Current().(T)
	if !ok {
		t.Fatalf("current scene is %T", h.mgr.Current())
	}
	return s
}

func TestTitleToOptionsAndBack(t *testing.T) {
	h := newHarness(t, NewTitle())
	h.press(ebiten.KeyQ) // any key
	current[*MainMenu](t, h)

	h.press(ebiten.KeyArrowDown) // to Options
	h.press(ebiten.KeyEnter)
	opts := current[*Options](t, h)

	// Down to Reduced motion and toggle it. Where the cursor starts depends
	// on the preset: Scale is disabled when there is only one scale mode.
	for range len(opts.menu.Items) {
		if opts.menu.Items[opts.menu.Cursor].Label == "Reduced motion" {
			break
		}
		h.press(ebiten.KeyArrowDown)
	}
	if got := opts.menu.Items[opts.menu.Cursor].Label; got != "Reduced motion" {
		t.Fatalf("cursor on %q", got)
	}
	h.press(ebiten.KeyArrowRight)
	if !h.ctx.Settings.ReducedMotion || !h.ctx.TakeSettingsChanged() {
		t.Error("Reduced motion was not turned on and reported")
	}

	h.press(ebiten.KeyEscape)
	current[*MainMenu](t, h)
	h.press(ebiten.KeyEscape)
	current[*Title](t, h)
}

func TestVolumeSteps(t *testing.T) {
	h := newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ticks(1) // builds the items
	opts := current[*Options](t, h)
	h.ctx.Settings.MusicVolume = 0.8
	for opts.menu.Items[opts.menu.Cursor].Label != "Music volume" {
		h.press(ebiten.KeyArrowDown)
	}
	h.press(ebiten.KeyArrowRight)
	h.press(ebiten.KeyArrowRight)
	h.press(ebiten.KeyArrowRight) // stops at the top
	if h.ctx.Settings.MusicVolume != 1 {
		t.Errorf("music volume %v, want 1", h.ctx.Settings.MusicVolume)
	}
	if got := opts.menu.Text(opts.menu.Cursor); got != "Music volume: 100%" {
		t.Errorf("shows %q", got)
	}
}

func TestMusicSwitchDimsItsSlider(t *testing.T) {
	h := newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ticks(1)
	opts := current[*Options](t, h)
	set := h.ctx.Settings
	label := func() string { return opts.menu.Items[opts.menu.Cursor].Label }
	for label() != "Music" {
		h.press(ebiten.KeyArrowDown)
	}
	h.press(ebiten.KeyEnter)
	if set.MusicOn || opts.menu.Text(opts.menu.Cursor) != "Music: Off" {
		t.Fatalf("Enter did not turn the music off: %q", opts.menu.Text(opts.menu.Cursor))
	}
	if !set.SFXOn {
		t.Error("turning the music off turned the effects off")
	}
	h.press(ebiten.KeyArrowDown)
	if label() != "Effects" {
		t.Errorf("Down from Music Off went to %q, want the slider skipped", label())
	}
	h.press(ebiten.KeyArrowUp)
	h.press(ebiten.KeyEnter) // music back on
	h.press(ebiten.KeyArrowDown)
	if !set.MusicOn || label() != "Music volume" {
		t.Errorf("with music on, Down went to %q (music on: %v)", label(), set.MusicOn)
	}
}

func TestSkillSelectStartsAGame(t *testing.T) {
	h := newHarness(t, NewMainMenu())
	h.press(ebiten.KeyEnter) // Play
	current[*SkillSelect](t, h)
	h.press(ebiten.KeyEnter) // the first skill
	current[*stubPlay](t, h)
}

func TestPlayDisabledWithoutAGame(t *testing.T) {
	h := newHarness(t, NewMainMenu())
	h.ctx.NewPlay = nil
	h.ticks(1)
	m := current[*MainMenu](t, h)
	if !m.menu.Items[0].Disabled || m.menu.Cursor != 1 {
		t.Errorf("Play should be disabled and skipped; cursor %d", m.menu.Cursor)
	}
}

func TestPauseAndResume(t *testing.T) {
	play := &stubPlay{}
	h := newHarness(t, play)
	h.in.Steps = append(h.in.Steps, input.Press(ebiten.KeyP), input.Idle(1))
	h.ticks(2)
	current[*Pause](t, h) // a cut: no fade
	before := play.updates
	h.ticks(10)
	if play.updates != before {
		t.Error("the paused scene kept running")
	}
	h.in.Steps = append(h.in.Steps, input.Press(ebiten.KeyEscape), input.Idle(1))
	h.ticks(2)
	if current[*stubPlay](t, h) != play {
		t.Error("resume did not return to the same scene")
	}
}

func TestQuitFromMenu(t *testing.T) {
	h := newHarness(t, NewMainMenu())
	h.press(ebiten.KeyArrowUp) // wraps from Play to Quit
	m := current[*MainMenu](t, h)
	if got := m.menu.Items[m.menu.Cursor].Label; got != "Quit" {
		t.Fatalf("cursor on %q, want Quit", got)
	}
	h.press(ebiten.KeyEnter)
	if !h.ctx.Quitting() {
		t.Error("Quit did not ask the game to close")
	}
}

func TestFadeDarkness(t *testing.T) {
	m := NewManager(NewTitle())
	if m.Darkness() != 1 {
		t.Errorf("starts at %v, want fully dark", m.Darkness())
	}
	set := config.DefaultSettings(config.Project)
	ctx := &Context{Input: input.New(&input.Script{}), Settings: &set}
	for range FadeTicks {
		ctx.Input.Update()
		m.Update(ctx)
	}
	if m.Darkness() != 0 {
		t.Errorf("after the fade-in %v, want 0", m.Darkness())
	}
}

func TestShake(t *testing.T) {
	var s Shaker
	s.Start(10, 3)
	if dx, _ := s.Offset(); dx != -3 && dx != 3 {
		t.Errorf("offset %d, want ±3", dx)
	}
	var signs []int
	for range 10 {
		s.update(false)
		dx, _ := s.Offset()
		signs = append(signs, dx)
	}
	if signs[0] == signs[1] || signs[len(signs)-1] != 0 {
		t.Errorf("offsets %v: want alternating, ending at 0", signs)
	}
	s.Start(10, 3)
	s.update(true)
	if dx, dy := s.Offset(); dx != 0 || dy != 0 {
		t.Error("reduced motion must stop the shake")
	}
}

func TestSounds(t *testing.T) {
	if err := audio.Join(Sounds).Check(); err != nil {
		t.Error(err)
	}
}

func TestTitleTuneAndMenuSounds(t *testing.T) {
	h := newHarness(t, NewTitle())
	h.ctx.Audio = audio.NewSynth(config.Chip4)
	h.ticks(1)
	if h.ctx.Audio.Tune() != TitleTune {
		t.Fatal("the title does not play the title tune")
	}
	h.press(ebiten.KeyQ)
	m := current[*MainMenu](t, h)
	if m.menu.OnMove == nil || m.menu.OnConfirm == nil {
		t.Error("the main menu has no sounds")
	}
	h.ctx.Audio.StopTune() // as a game does
	h.press(ebiten.KeyEscape)
	h.press(ebiten.KeyQ)
	if h.ctx.Audio.Tune() != TitleTune {
		t.Error("returning to the menus does not restart the title tune")
	}
}

func TestSoundTest(t *testing.T) {
	h := newHarness(t, NewSoundTest(func() Scene { return NewMainMenu() }))
	h.ctx.Audio = audio.NewSynth(config.Chip4)
	h.ctx.Sounds = Sounds
	h.ticks(1)
	st := current[*SoundTest](t, h)
	if n := len(st.menu.Items); n != len(Sounds.Tunes)+len(Sounds.Effects)+2 {
		t.Fatalf("%d items", n)
	}
	h.press(ebiten.KeyEnter) // the first tune
	if h.ctx.Audio.Tune() != Sounds.Tunes[0] {
		t.Error("the sound test did not play the first tune")
	}
	h.in.Steps = append(h.in.Steps, input.Press(ebiten.KeyEscape), input.Idle(1))
	h.ticks(2)
	if h.ctx.Audio.Tune() != nil {
		t.Error("leaving the sound test did not stop the sound")
	}
}

// TestLightOption checks the Light option starts at Smooth, switches to
// Pixel and back, and reports the change for saving.
func TestLightOption(t *testing.T) {
	h := newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ticks(1)
	opts := current[*Options](t, h)
	for range len(opts.menu.Items) {
		if opts.menu.Items[opts.menu.Cursor].Label == "Light" {
			break
		}
		h.press(ebiten.KeyArrowDown)
	}
	if got := opts.menu.Text(opts.menu.Cursor); got != "Light: Smooth" {
		t.Fatalf("shows %q, want Light: Smooth by default", got)
	}
	h.press(ebiten.KeyArrowRight)
	if h.ctx.Settings.SmoothLight || !h.ctx.TakeSettingsChanged() || opts.menu.Text(opts.menu.Cursor) != "Light: Pixel" {
		t.Errorf("after Right: smooth %v, shows %q", h.ctx.Settings.SmoothLight, opts.menu.Text(opts.menu.Cursor))
	}
	h.press(ebiten.KeyArrowLeft)
	if !h.ctx.Settings.SmoothLight {
		t.Error("Left did not switch back to Smooth")
	}
}

// TestGameTunes checks the title plays the game's own title tune when it
// has one, and the high score table its high score tune.
func TestGameTunes(t *testing.T) {
	title := &audio.Tune{Name: "own title", Parts: []audio.Part{{Notes: audio.Notes("C4:12")}}}
	scores := &audio.Tune{Name: "own scores", Parts: []audio.Part{{Notes: audio.Notes("E4:12")}}}
	h := newHarness(t, NewTitle())
	h.ctx.Audio = audio.NewSynth(config.Chip4)
	h.ctx.TitleTune, h.ctx.ScoresTune = title, scores
	h.ticks(1)
	if got := h.ctx.Audio.Tune(); got != title {
		t.Errorf("title plays %v, want the game's own", got)
	}
	h = newHarness(t, NewHighScoreTable(config.Skills[0], -1))
	h.ctx.Audio = audio.NewSynth(config.Chip4)
	h.ctx.TitleTune, h.ctx.ScoresTune = title, scores
	h.ticks(1)
	if got := h.ctx.Audio.Tune(); got != scores {
		t.Errorf("high score table plays %v, want the game's own", got)
	}
}
