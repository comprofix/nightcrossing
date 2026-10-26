package scene

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/score"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// harness runs scenes headless: Update only, never Draw.
type harness struct {
	t   *testing.T
	ctx *Context
	mgr *Manager
	in  *input.Script
}

func newHarness(t *testing.T, first Scene) *harness {
	set := config.DefaultSettings(config.ProjectLooks())
	script := &input.Script{}
	h := &harness{
		t:   t,
		in:  script,
		mgr: NewManager(first),
		ctx: &Context{
			Input:    input.New(script),
			RNG:      rand.New(rand.NewPCG(1, 2)),
			Preset:   config.Project,
			Looks:    config.ProjectLooks(),
			Settings: &set,
			NewPlay:  func(*Context, config.Mode) Scene { return &stubPlay{} },
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

// pad presses gamepad buttons for a tick, like press.
func (h *harness) pad(buttons ...input.Button) {
	h.in.Steps = append(h.in.Steps, input.Step{Buttons: buttons, Ticks: 1}, input.Idle(1))
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

func TestModeSelectStartsAGame(t *testing.T) {
	h := newHarness(t, NewMainMenu())
	h.press(ebiten.KeyEnter) // Play
	current[*ModeSelect](t, h)
	h.press(ebiten.KeyEnter) // the first mode
	current[*Intro](t, h)
	h.press(ebiten.KeyEnter) // play it
	current[*stubPlay](t, h)
}

func TestIntroBackReturnsToItsMode(t *testing.T) {
	h := newHarness(t, NewIntro(0))
	h.press(ebiten.KeyEscape)
	if s := current[*ModeSelect](t, h); s.menu.Cursor != 0 {
		t.Errorf("cursor on mode %d, want 0", s.menu.Cursor)
	}
}

func TestSuggestedMode(t *testing.T) {
	tab := score.New()
	if n := SuggestedMode(config.Modes, tab); n != -1 {
		t.Errorf("nothing mastered: suggested %d, want none", n)
	}
	tab.MarkMastered(config.Modes[0].ID)
	if n := SuggestedMode(config.Modes, tab); n != 1 {
		t.Errorf("first mode mastered: suggested %d, want the one it opened, 1", n)
	}
	for _, m := range config.Modes {
		tab.MarkMastered(m.ID)
	}
	if n := SuggestedMode(config.Modes, tab); n != -1 {
		t.Errorf("every mode mastered: suggested %d, want none", n)
	}
}

// TestModeSelectLocksAndMarks: a mastered mode is marked, the one it opened
// is suggested, and the modes after are dimmed with what opens them and
// can't be chosen (D64).
func TestModeSelectLocksAndMarks(t *testing.T) {
	h := newHarness(t, NewModeSelect())
	h.ctx.Scores.MarkMastered(config.Modes[0].ID)
	h.ticks(1)
	m := current[*ModeSelect](t, h).menu
	if l := m.Items[0].Label; !strings.HasSuffix(l, Tick+" MASTERED") {
		t.Errorf("label %q, want it marked mastered", l)
	}
	if l := m.Items[1].Label; !strings.HasSuffix(l, "NEXT") || m.Items[1].Disabled || m.Cursor != 1 {
		t.Errorf("label %q, disabled %v, cursor %d: want the opened mode suggested", l, m.Items[1].Disabled, m.Cursor)
	}
	want := Lock + " MASTER " + strings.ToUpper(config.Modes[1].Name) + " TO OPEN"
	if l := m.Items[2].Label; !strings.HasSuffix(l, want) || !m.Items[2].Disabled {
		t.Errorf("label %q, disabled %v: want it locked, %q", l, m.Items[2].Disabled, want)
	}
}

// TestTestGamesCanPickALockedMode: a test game can start any mode (D64).
func TestTestGamesCanPickALockedMode(t *testing.T) {
	h := newHarness(t, NewModeSelect())
	h.ctx.StartLevel = 5
	h.ticks(1)
	for i, it := range current[*ModeSelect](t, h).menu.Items {
		if it.Disabled {
			t.Errorf("mode %d disabled in a test game", i)
		}
	}
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
	set := config.DefaultSettings(config.ProjectLooks())
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
	title := &audio.Tune{Name: "own title", Loop: true, Parts: []audio.Part{{Notes: audio.Notes("C4:12")}}}
	h.ctx.TitleTune = title
	h.ticks(1)
	if h.ctx.Audio.Tune() != title {
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
	if h.ctx.Audio.Tune() != title {
		t.Error("returning to the menus does not restart the title tune")
	}
}

func TestSoundTest(t *testing.T) {
	h := newHarness(t, NewSoundTest(func() Scene { return NewMainMenu() }))
	h.ctx.Audio = audio.NewSynth(config.Chip4)
	tune := &audio.Tune{Name: "a tune", Parts: []audio.Part{{Notes: audio.Notes("C4:12")}}}
	bank := audio.Join(audio.Bank{Tunes: []*audio.Tune{tune}}, Sounds)
	h.ctx.Sounds = bank
	h.ticks(1)
	st := current[*SoundTest](t, h)
	if n := len(st.menu.Items); n != len(bank.Tunes)+len(bank.Effects)+2 {
		t.Fatalf("%d items", n)
	}
	h.press(ebiten.KeyEnter) // the first tune
	if h.ctx.Audio.Tune() != tune {
		t.Error("the sound test did not play the first tune")
	}
	h.in.Steps = append(h.in.Steps, input.Press(ebiten.KeyEscape), input.Idle(1))
	h.ticks(2)
	if h.ctx.Audio.Tune() != nil {
		t.Error("leaving the sound test did not stop the sound")
	}
}

// TestLightOption checks the Light option starts at Smooth, switches to
// Pixel and back, and reports the change for saving. It is the pixel
// look's (D80).
func TestLightOption(t *testing.T) {
	h := newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ctx.Settings.Look = config.Modern.Name
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
	h = newHarness(t, NewHighScoreTable(config.Modes[0], -1))
	h.ctx.Audio = audio.NewSynth(config.Chip4)
	h.ctx.TitleTune, h.ctx.ScoresTune = title, scores
	h.ticks(1)
	if got := h.ctx.Audio.Tune(); got != scores {
		t.Errorf("high score table plays %v, want the game's own", got)
	}
}

func TestOptionsWithAGamepad(t *testing.T) {
	h := newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ticks(1)
	opts := current[*Options](t, h)
	for opts.menu.Items[opts.menu.Cursor].Label != "High contrast" {
		h.pad(input.PadDown)
	}
	h.pad(input.PadRight)
	if !h.ctx.Settings.HighContrast {
		t.Error("Right on the pad did not turn high contrast on")
	}
	h.pad(input.PadB)
	current[*MainMenu](t, h)
}

func TestShaderOptionsWithoutShaders(t *testing.T) {
	h := newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ctx.NoShaders = true
	h.ctx.Settings.SmoothLight, h.ctx.Settings.Glow, h.ctx.Settings.CRT = false, true, true
	h.ticks(1)
	opts := current[*Options](t, h)
	want := map[string]string{"Light": "Light: Smooth", "Glow": "Glow: Off", "CRT": "CRT: Off"}
	for i, it := range opts.menu.Items {
		if w, ok := want[it.Label]; ok {
			if !it.IsDisabled() || opts.menu.Text(i) != w {
				t.Errorf("%s: disabled %v, shows %q, want dimmed and %q", it.Label, it.IsDisabled(), opts.menu.Text(i), w)
			}
			delete(want, it.Label)
		}
	}
	if len(want) > 0 {
		t.Errorf("missing items %v", want)
	}
}

// TestModesInProgressInTheMenus: a mode added to the context's list (as
// -mode does in debug builds) shows in the MODE menu and its intro plays it.
func TestModesInProgressInTheMenus(t *testing.T) {
	extra := config.Mode{ID: "extra", Name: "Extra", Desc: "a mode in progress", Levels: config.StrollLevels}
	var played string
	h := newHarness(t, NewModeSelect())
	h.ctx.Modes = append(append([]config.Mode(nil), config.Modes...), extra)
	h.ctx.NewPlay = func(_ *Context, m config.Mode) Scene { played = m.ID; return &stubPlay{} }
	h.ticks(1)
	s := current[*ModeSelect](t, h)
	if n := len(s.menu.Items); n != len(config.Modes)+1 {
		t.Fatalf("%d modes in the menu", n)
	}
	for range len(config.Modes) {
		h.press(ebiten.KeyArrowDown)
	}
	h.press(ebiten.KeyEnter)
	h.press(ebiten.KeyEnter)
	if played != "extra" {
		t.Errorf("played %q, want extra", played)
	}
}

// TestOptionsLook checks Look comes first in Options only in a game with two
// looks, switches between them, and that Scale is dimmed while the shown
// look has one mode, and Light and CRT under the smooth look (D80).
func TestOptionsLook(t *testing.T) {
	one := config.Looks{First: config.Modern}
	set := config.DefaultSettings(one)
	h := newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ctx.Looks, h.ctx.Settings = one, &set
	h.ticks(1)
	if l := current[*Options](t, h).menu.Items[0].Label; l == "Look" {
		t.Error("a game with one look offers Look")
	}

	modern := config.Modern // given CRT, so Scale has two modes in it
	modern.ScaleModes = []config.ScaleMode{config.ScaleInteger, config.ScaleCRT}
	looks := config.Looks{First: config.Smooth, Second: &modern}
	set = config.DefaultSettings(looks)
	set.CRT = true
	h = newHarness(t, NewOptions(func() Scene { return NewMainMenu() }))
	h.ctx.Looks, h.ctx.Settings = looks, &set
	h.ticks(1)
	m := &current[*Options](t, h).menu
	if m.Items[0].Label != "Look" || m.Text(0) != "Look: Smooth" {
		t.Fatalf("first item %q, want Look on Smooth", m.Text(0))
	}
	item := func(label string) (ui.Item, string) {
		for i, it := range m.Items {
			if it.Label == label {
				return it, m.Text(i)
			}
		}
		t.Fatalf("no %s in Options", label)
		return ui.Item{}, ""
	}
	pixelOnly := func(look string, dimmed bool, light, crt string) {
		t.Helper()
		for _, c := range []struct{ label, want string }{{"Scale", ""}, {"Light", light}, {"CRT", crt}} {
			it, text := item(c.label)
			if it.IsDisabled() != dimmed {
				t.Errorf("%s look: %s dimmed %v, want %v", look, c.label, it.IsDisabled(), dimmed)
			}
			if c.want != "" && text != c.label+": "+c.want {
				t.Errorf("%s look: %q, want %s: %s", look, text, c.label, c.want)
			}
		}
	}
	pixelOnly("smooth", true, "Smooth", "Off")
	h.press(ebiten.KeyEnter)
	if set.Look != modern.Name || !h.ctx.TakeSettingsChanged() {
		t.Fatalf("Enter on Look left it on %q", set.Look)
	}
	set.SmoothLight = false
	pixelOnly("pixel", false, "Pixel", "On")
	h.press(ebiten.KeyArrowRight)
	if set.Look != config.Smooth.Name {
		t.Errorf("Right on Look left it on %q, want back to Smooth", set.Look)
	}
}

// lookPlay is a play scene that notes the screen width it is made and
// updated at.
type lookPlay struct {
	stubPlay
	made, updated int
}

func (p *lookPlay) Update(ctx *Context) error {
	p.updated = ctx.Preset.ScreenWidth
	p.updates++
	if ctx.Input.JustPressed(input.Pause) {
		ctx.Cut(NewPause(p)) // itself, so it is what resumes
	}
	return nil
}

// nativePlay is a lookPlay redrawn for the smooth look.
type nativePlay struct{ lookPlay }

func (*nativePlay) Native() {}

// TestPixelLookPlay checks that while the smooth look of two is shown, a
// play scene that is not Native is made and updated in the pixel look, the
// shared scenes stay in the smooth look, and what the play scene asks of
// the context (here a pause) is kept.
func TestPixelLookPlay(t *testing.T) {
	modern := config.Modern
	h := newHarness(t, NewModeSelect())
	h.ctx.Looks = config.Looks{First: config.Smooth, Second: &modern}
	h.ctx.Preset = config.Smooth
	var play *lookPlay
	h.ctx.NewPlay = func(ctx *Context, _ config.Mode) Scene {
		play = &lookPlay{made: ctx.Preset.ScreenWidth}
		return play
	}
	h.press(ebiten.KeyEnter) // the first mode
	current[*Intro](t, h)
	h.press(ebiten.KeyEnter) // play it
	if play == nil || play.made != modern.ScreenWidth || play.updated != modern.ScreenWidth {
		t.Fatalf("play made and updated at %+v, want the pixel look's width %d", play, modern.ScreenWidth)
	}
	if h.ctx.Preset.Name != config.Smooth.Name {
		t.Errorf("the context was left in %s", h.ctx.Preset.Name)
	}
	h.in.Steps = append(h.in.Steps, input.Press(ebiten.KeyP), input.Idle(1))
	h.ticks(2)
	current[*Pause](t, h)

	// A Native play scene is made in the smooth look, and must be said to be.
	h = newHarness(t, NewIntro(0))
	h.ctx.Looks = config.Looks{First: config.Smooth, Second: &modern}
	h.ctx.Preset = config.Smooth
	var native *nativePlay
	h.ctx.NewPlay = func(ctx *Context, _ config.Mode) Scene {
		native = &nativePlay{lookPlay{made: ctx.Preset.ScreenWidth}}
		return native
	}
	h.ctx.NativePlay = true
	h.press(ebiten.KeyEnter)
	if native == nil || native.made != config.Smooth.ScreenWidth || native.updated != config.Smooth.ScreenWidth {
		t.Errorf("native play made and updated at %+v, want the smooth look's width", native)
	}
	h.ctx.NativePlay = false
	defer func() {
		if recover() == nil {
			t.Error("a Native play scene without NativePlay did not panic")
		}
	}()
	h.ctx.makeGame(func(c *Context) Scene { return c.NewPlay(c, config.Modes[0]) })
}

// TestLookDuringPlay switches the look from Options on the pause screen,
// both ways, and checks the game resumed under it is still updated in the
// pixel look while the context is left in the look chosen.
func TestLookDuringPlay(t *testing.T) {
	modern := config.Modern
	looks := config.Looks{First: config.Smooth, Second: &modern}
	set := config.DefaultSettings(looks)
	set.Look = modern.Name
	h := newHarness(t, NewModeSelect())
	h.ctx.Looks, h.ctx.Settings, h.ctx.Preset = looks, &set, modern
	var play *lookPlay
	h.ctx.NewPlay = func(ctx *Context, _ config.Mode) Scene {
		play = &lookPlay{made: ctx.Preset.ScreenWidth}
		return play
	}
	h.press(ebiten.KeyEnter) // the first mode
	h.press(ebiten.KeyEnter) // play it
	if play == nil || play.made != modern.ScreenWidth {
		t.Fatalf("play made at %+v, want the pixel look's width %d", play, modern.ScreenWidth)
	}

	for _, look := range []config.Preset{config.Smooth, modern} {
		h.press(ebiten.KeyP)
		current[*Pause](t, h)
		h.press(ebiten.KeyArrowDown)
		h.press(ebiten.KeyEnter) // Options
		h.press(ebiten.KeyEnter) // Look, the first item
		if set.Look != look.Name || !h.ctx.TakeSettingsChanged() {
			t.Fatalf("Look in Options from the pause screen left it on %q, want %s", set.Look, look.Name)
		}
		h.ctx.Preset = set.Shown(looks) // as the game shows the look chosen
		h.press(ebiten.KeyEscape)
		current[*Pause](t, h)
		h.press(ebiten.KeyP)
		play.updated = 0
		h.ticks(1)
		if current[*lookPlay](t, h) != play {
			t.Fatal("the game resumed is not the one paused")
		}
		if play.updated != modern.ScreenWidth {
			t.Errorf("under %s, play updated at width %d, want the pixel look's %d", look.Name, play.updated, modern.ScreenWidth)
		}
		if h.ctx.Preset.Name != look.Name {
			t.Errorf("the context was left in %s, want %s", h.ctx.Preset.Name, look.Name)
		}
	}
}

// click clicks the middle of item i of a menu screen's menu, lets go, and
// waits out any fade.
func (h *harness) click(i int, top func(*Context) int) {
	f := h.ctx.Fonts
	y := top(h.ctx) + i*(f.Small.LineHeight()+ui.LineGap) + f.Small.LineHeight()/2
	h.mouse(input.Click(h.ctx.Preset.ScreenWidth/2, y))
}

// mouse plays a one-tick step, lets go, and waits out any fade.
func (h *harness) mouse(st input.Step) {
	h.in.Steps = append(h.in.Steps, st, input.Idle(1))
	h.ticks(2)
	h.ticks(2 * FadeTicks)
}

// rightClick clicks the right mouse button.
func (h *harness) rightClick() {
	h.mouse(input.Step{Mouse: []ebiten.MouseButton{ebiten.MouseButtonRight}, Ticks: 1})
}

// TestMenusByMouse: the shared scenes can be used by mouse alone (D81).
func TestMenusByMouse(t *testing.T) {
	h := newHarness(t, NewTitle())
	h.ctx.Fonts = gfx.SmoothFonts(h.ctx.Preset.ScreenHeight) // the mouse needs the menus measured
	h.click(0, menuTop)                                      // any button
	current[*MainMenu](t, h)
	h.click(1, menuTop)
	opts := current[*Options](t, h)

	// The Reduced motion line, clicked, toggles it.
	for i, it := range opts.menu.Items {
		if it.Label == "Reduced motion" {
			h.click(i, menuTop)
		}
	}
	if !h.ctx.Settings.ReducedMotion {
		t.Error("Reduced motion was not turned on by a click")
	}
	h.rightClick() // back
	current[*MainMenu](t, h)
	h.click(0, menuTop) // Play
	h.click(0, menuTop) // the first mode
	current[*Intro](t, h)
	h.click(0, menuTop) // anywhere plays it
	play := current[*stubPlay](t, h)
	h.rightClick() // pause
	current[*Pause](t, h)
	h.click(0, pauseMenuTop) // Resume
	if current[*stubPlay](t, h) != play {
		t.Error("Resume did not return to the game")
	}
}

// TestLookByMouse switches the look by clicks alone, both ways from the
// main menu's Options and to the smooth look from a paused game's, which
// resumes still in the pixel look.
func TestLookByMouse(t *testing.T) {
	modern := config.Modern
	looks := config.Looks{First: config.Smooth, Second: &modern}
	set := config.DefaultSettings(looks)
	set.Look = modern.Name
	h := newHarness(t, NewMainMenu())
	h.ctx.Looks, h.ctx.Settings = looks, &set
	var play *lookPlay
	h.ctx.NewPlay = func(ctx *Context, _ config.Mode) Scene {
		play = &lookPlay{made: ctx.Preset.ScreenWidth}
		return play
	}
	// show shows the look the settings choose, as the game does, with fonts
	// to measure the menus by.
	show := func() {
		h.ctx.TakeSettingsChanged()
		h.ctx.Preset = set.Shown(looks)
		h.ctx.Fonts = gfx.SmoothFonts(h.ctx.Preset.ScreenHeight)
	}
	clickLook := func(want config.Preset) {
		t.Helper()
		h.click(0, menuTop) // Look, the first item
		if set.Look != want.Name {
			t.Fatalf("a click on Look left it on %q, want %s", set.Look, want.Name)
		}
		show()
	}
	show()
	h.click(1, menuTop) // Options
	current[*Options](t, h)
	clickLook(config.Smooth)
	clickLook(modern)
	h.rightClick() // back
	current[*MainMenu](t, h)

	h.click(0, menuTop) // Play
	h.click(0, menuTop) // the first mode
	h.click(0, menuTop) // anywhere on the intro plays it
	if play == nil || play.made != modern.ScreenWidth {
		t.Fatalf("play made at %+v, want the pixel look's width %d", play, modern.ScreenWidth)
	}
	h.rightClick() // pause
	current[*Pause](t, h)
	h.click(1, pauseMenuTop) // Options
	current[*Options](t, h)
	clickLook(config.Smooth)
	h.rightClick() // back
	current[*Pause](t, h)
	h.click(0, pauseMenuTop) // Resume
	play.updated = 0
	h.ticks(1)
	if current[*lookPlay](t, h) != play {
		t.Fatal("the game resumed is not the one paused")
	}
	if play.updated != modern.ScreenWidth || h.ctx.Preset.Name != config.Smooth.Name {
		t.Errorf("play updated at width %d with the context in %s, want %d in Smooth",
			play.updated, h.ctx.Preset.Name, modern.ScreenWidth)
	}
}
