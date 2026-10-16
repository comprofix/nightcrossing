# Night Crossing

Created from [gameskeleton](https://github.com/comprofix/gameskeleton) v0.1.0.

## Requirements

- Go 1.27 or later
- **Linux only:** cgo plus the X11/GL/ALSA development packages:
  `libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config`

## Build and run

```sh
go run ./cmd/nightcrossing                 # run the game
go run -tags debug ./cmd/nightcrossing     # with the debug overlay
go generate ./assets               # rebuild sprite sheets
go vet ./...
go test ./...
go build -o bin/ ./cmd/nightcrossing
```

## Personal documents

`DESIGN.md` and `PLAN.md` are personal working documents. `.gitignore` keeps
them out of git, so back them up somewhere else.

## License

MIT. See [LICENSE](LICENSE).
