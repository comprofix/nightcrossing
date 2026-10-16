// Command newgame turns a fresh copy of the skeleton into a new game.
// It does the steps that must be exact:
//
//	go run ./tools/newgame -module github.com/comprofix/<game> -name <game> -title "<Game Title>" [-keep-demo]
//
// It changes the module path in go.mod and every import, and the skeleton's
// name in code; renames cmd/gameskeleton to cmd/<game>;
// sets config.Title; removes the demo unless -keep-demo is given; writes a
// new README saying which skeleton version it came from; removes itself;
// and runs go build, go vet and go test to check everything still works.
//
// It refuses to run where DESIGN.md exists, so it can't damage the skeleton
// itself, whose working documents live beside it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SkeletonVersion is the skeleton release this copy is. Keep it in step with
// the release tag.
const SkeletonVersion = "0.1.0"

const (
	oldModule = "github.com/comprofix/gameskeleton"
	oldName   = "gameskeleton"
)

// The demo's files and folders, relative to the root.
var demoPaths = []string{
	"internal/demo",
	"internal/game/demo.go",
	"assets/sprites/demo.txt",
	"assets/gen/demo.png",
	"assets/gen/demo.json",
}

// Folders the rewrite never enters.
var skipDirs = map[string]bool{".git": true, ".claude": true, "bin": true, "dist": true, "ref": true, "preview": true}

type options struct {
	dir, module, name, title string
	keepDemo, check          bool
}

func main() {
	var o options
	flag.StringVar(&o.dir, "dir", ".", "the copy of the skeleton to change")
	flag.StringVar(&o.module, "module", "", "the new module path, e.g. github.com/comprofix/mygame")
	flag.StringVar(&o.name, "name", "", "the command's name, e.g. mygame (lower case letters, digits and -)")
	flag.StringVar(&o.title, "title", "", `the game's title, e.g. "My Game"`)
	flag.BoolVar(&o.keepDemo, "keep-demo", false, "keep the Dot Dash demo")
	flag.BoolVar(&o.check, "check", true, "run go build and go test afterwards")
	flag.Parse()
	if err := run(o, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "newgame:", err)
		os.Exit(1)
	}
}

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func run(o options, log io.Writer) error {
	switch {
	case o.module == "" || strings.ContainsAny(o.module, " \t\"") || o.module == oldModule:
		return fmt.Errorf("-module %q: give the new game's module path", o.module)
	case !validName.MatchString(o.name) || o.name == oldName:
		return fmt.Errorf("-name %q: use lower case letters, digits and -, starting with a letter", o.name)
	case strings.TrimSpace(o.title) == "":
		return errors.New("-title: give the game's title")
	}
	root := o.dir
	if _, err := os.Stat(filepath.Join(root, "DESIGN.md")); err == nil {
		return errors.New("DESIGN.md exists here: newgame runs on a fresh copy before DESIGN.md is written (move it aside to go on)")
	}
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(gomod), "module "+oldModule+"\n") {
		return errors.New("go.mod is not the skeleton's: has this copy already been set up?")
	}

	// Remove first, so nothing is rewritten only to be deleted.
	remove := []string{"tools/newgame"}
	if !o.keepDemo {
		remove = append(remove, demoPaths...)
	}
	for _, p := range remove {
		if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			return err
		}
		fmt.Fprintln(log, "removed", p)
	}

	n, err := rewrite(root, o)
	if err != nil {
		return err
	}
	fmt.Fprintf(log, "renamed the module and %s in %d files\n", oldName, n)

	if err := os.Rename(filepath.Join(root, "cmd", oldName), filepath.Join(root, "cmd", o.name)); err != nil {
		return err
	}
	fmt.Fprintf(log, "moved cmd/%s to cmd/%s\n", oldName, o.name)

	if err := setTitle(filepath.Join(root, "internal", "config", "project.go"), o.title); err != nil {
		return err
	}
	fmt.Fprintf(log, "set the title to %q\n", o.title)

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(readme(o)), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(log, "wrote README.md")

	if o.check {
		for _, args := range [][]string{{"build", "./..."}, {"vet", "./..."}, {"test", "./..."}} {
			fmt.Fprintln(log, "go", strings.Join(args, " "))
			cmd := exec.Command("go", args...)
			cmd.Dir, cmd.Stdout, cmd.Stderr = root, log, log
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go %s: %w", args[0], err)
			}
		}
	}
	fmt.Fprintln(log, "done")
	return nil
}

// rewrite changes the module path, then the skeleton's name, in every Go
// file, go.mod and Markdown file, and returns how many it changed.
func rewrite(root string, o options) (int, error) {
	changed := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".mod", ".md":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := strings.ReplaceAll(string(data), oldModule, o.module)
		s = strings.ReplaceAll(s, oldName, o.name)
		if s == string(data) {
			return nil
		}
		changed++
		return os.WriteFile(path, []byte(s), 0o644)
	})
	return changed, err
}

var titleLine = regexp.MustCompile(`(?m)^const Title = ".*"$`)

// setTitle sets config.Title in project.go.
func setTitle(path, title string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !titleLine.Match(data) {
		return fmt.Errorf("%s: no `const Title = \"...\"` line", path)
	}
	out := titleLine.ReplaceAllLiteral(data, []byte("const Title = "+strconv.Quote(title)))
	return os.WriteFile(path, out, 0o644)
}

// readme is the new game's starting README.
func readme(o options) string {
	return fmt.Sprintf(`# %s

Created from [gameskeleton](https://github.com/comprofix/gameskeleton) v%s.

## Requirements

- Go 1.27 or later
- **Linux only:** cgo plus the X11/GL/ALSA development packages:
  `+"`libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config`"+`

## Build and run

`+"```sh"+`
go run ./cmd/%[3]s                 # run the game
go run -tags debug ./cmd/%[3]s     # with the debug overlay
go generate ./assets               # rebuild sprite sheets
go vet ./...
go test ./...
go build -o bin/ ./cmd/%[3]s
`+"```"+`

## Personal documents

`+"`DESIGN.md` and `PLAN.md`"+` are personal working documents. `+"`.gitignore`"+` keeps
them out of git, so back them up somewhere else.

## License

MIT. See [LICENSE](LICENSE).
`, o.title, SkeletonVersion, o.name)
}
