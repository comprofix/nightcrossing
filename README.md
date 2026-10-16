# Game Skeleton

A starter template for retro and arcade games written in Go with
[Ebitengine](https://ebitengine.org/). New games are made from it with
GitHub's **Use this template** button. It comes with a tiny demo game,
**Dot Dash**, which shows every part working and is removed when a real game
begins.

## What you get

- **Display:** a fixed internal resolution scaled up by whole numbers with
  letterboxing, fullscreen, and an optional CRT 4:3 mode. One palette per
  game, with colours named in code (`resurrect64.Gold`).
- **Art:** text-grid sprites in `assets/sprites/*.txt`, packed into sprite
  sheets by `go generate`, with a preview sheet for checking them by eye. An
  8×8 font and a 16×16 font made from it.
- **Scenes:** title, main menu, skill select, options, pause, tally, high
  score entry and table, with fades and screen shake.
- **Input:** keyboard and gamepad mapped to actions, with key repeat for menus.
- **Sound made in code:** a chip synth with pulse, triangle and noise waves and
  volume envelopes, tunes and effects as note tables, timed to the game tick.
- **Saves:** settings and a top-10 high score table for each skill.
- **Testing:** game logic runs headless and deterministic from a seed;
  scripted input, a scene harness and golden replay tests. A debug build shows
  a timing overlay and collision boxes.

## Requirements

- Go 1.27 or later
- **Linux only:** cgo plus the X11/GL/ALSA development packages:
  `libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config`

## Presets

A game picks one preset in `internal/config/project.go`, along with its title
and tick rate. Nothing else in the code hard-codes a screen size or palette.

| | Modern (default) | Classic PC |
|---|---|---|
| Resolution | 640×360 (16:9) | 320×200 |
| Palette | Resurrect 64 (64 colours) | Sweetie 16 (16 colours) |
| Scaling | whole-number | whole-number, or CRT 4:3 |
| Sound | `Chip4`: 4 channels (2 pulse, triangle, noise) | `Speaker`: 1 square-wave voice |

A game can also add its own `Preset` in `internal/config/preset.go`. The game
runs at 60 ticks per second, and all game logic counts in ticks.

## Starting a new game

1. On GitHub, press **Use this template** to make `comprofix/<game>`, and clone
   it.
2. Run:

   ```sh
   go run ./tools/newgame -module github.com/comprofix/<game> -name <game> -title "<Game Title>"
   ```

   It changes the module path and every import, renames `cmd/gameskeleton` to
   `cmd/<game>`, sets the title, removes the demo (add `-keep-demo` to keep
   it), writes a new README, removes itself, and builds and tests the result.
3. Set the preset in `internal/config/project.go`.
4. Wire in your play scene, sounds and sprite sheets through `game.Content`,
   in a file such as `internal/game/<game>.go`. Until then, Play is greyed out
   in the menu.

## Commands

```sh
go run ./cmd/gameskeleton                 # run the game
go run -tags debug ./cmd/gameskeleton     # with the debug overlay and Tab test pattern
go run ./cmd/gameskeleton -seed 42        # with a fixed random seed
go generate ./assets                      # rebuild sprite sheets from assets/sprites/*.txt
go run ./tools/spritegen -in assets/sprites -out assets/gen -preview <dir>   # preview sheets
go vet ./...
go test ./...
go build -o bin/ ./cmd/gameskeleton
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/gameskeleton.exe ./cmd/gameskeleton
```

Linux builds need cgo, so build them on Linux or in WSL.

## Controls

| Action | Keyboard | Gamepad (standard layout) |
|---|---|---|
| Move | Arrow keys, WASD | D-pad, left stick |
| Confirm | Enter, Space | A |
| Back | Esc, Backspace | B |
| Pause | Esc, P | Start |
| Restart | Ctrl+R | — |
| Main menu | Ctrl+M | Back/Select |
| Sound on/off | Ctrl+S | — |
| Scale mode | F9 | — |
| Quit | F10 | — |
| Fullscreen | F11, Alt+Enter | — |

F9, F10, F11 and Alt+Enter are kept for the window and never reach the game.

## Saved files

Both are written next to the game (or in the working folder under `go run`),
atomically, so a crash can't leave half a file.

- `settings.json`: window size, fullscreen, scale mode, sound, music and
  effects switches, master, music and effects volumes, and reduced motion.
- `scores.json`: a top-10 table for each skill, with 3-letter initials.

A missing or broken file means defaults, never a crash, and every value is
checked when loaded.

## Testing

- Game logic never touches graphics or sound, so it runs in plain unit tests.
- `internal/scene/scenetest` runs scenes headless from a seed and a script of
  inputs. `scenetest.Golden` compares a result with `testdata/*.golden`;
  after an intended change, rewrite them with `go test ./<package> -update`.
- `go run -tags debug` adds an overlay with ticks and frames per second, the
  tick count and the scene, and lets scenes draw collision boxes.

## Keeping games and the skeleton in step

The skeleton is tagged by version (`v0.1.0`, …), and each new game's README
records the version it came from. A fix made in a game that would help every
game is copied back into the skeleton by hand and released as a new tag.
Existing games are not updated automatically.

## Personal documents

`DESIGN.md` and `PLAN.md` are personal working documents. `.gitignore` keeps
them out of git, so back them up somewhere else.

## License

MIT. See [LICENSE](LICENSE).
