package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// skeleton is the skeleton's root, two folders up.
const skeleton = "../.."

// copySkeleton copies what a fresh clone of the skeleton has into dst:
// everything but git's folder, local documents and settings, and build
// output.
func copySkeleton(t *testing.T, dst string) {
	t.Helper()
	skipFiles := map[string]bool{"DESIGN.md": true, "PLAN.md": true, "settings.json": true, "scores.json": true}
	err := filepath.WalkDir(skeleton, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(skeleton, path)
		if d.IsDir() {
			if rel != "." && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if skipFiles[rel] || filepath.Ext(rel) == ".exe" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

// leftovers lists every line still naming the skeleton, apart from the
// README's note of where the game came from.
func leftovers(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, oldName) && !(rel == "README.md" && strings.HasPrefix(line, "Created from")) {
				found = append(found, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(rel), i+1, line))
			}
		}
		return nil
	})
	return found
}

func TestNewGame(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and tests whole games")
	}
	for _, keep := range []bool{false, true} {
		name := "without the demo"
		if keep {
			name = "with the demo"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			copySkeleton(t, dir)
			var log bytes.Buffer
			o := options{dir: dir, module: "example.com/me/rocket-rumble", name: "rocket-rumble", title: `Rocket "RR" Rumble`, keepDemo: keep, check: true}
			if err := run(o, &log); err != nil {
				t.Fatalf("%v\n%s", err, log.String())
			}
			t.Log(log.String())

			for _, rel := range []string{"cmd/rocket-rumble/main.go", "internal/scene/title.go", "assets/gen/README.md"} {
				if !exists(dir, rel) {
					t.Errorf("%s is missing", rel)
				}
			}
			for _, rel := range []string{"cmd/gameskeleton", "tools/newgame"} {
				if exists(dir, rel) {
					t.Errorf("%s is still there", rel)
				}
			}
			for _, rel := range demoPaths {
				if exists(dir, rel) != keep {
					t.Errorf("%s: there %v, want %v", rel, !keep, keep)
				}
			}
			if gomod, _ := os.ReadFile(filepath.Join(dir, "go.mod")); !strings.HasPrefix(string(gomod), "module example.com/me/rocket-rumble\n") {
				t.Errorf("go.mod starts %q", strings.SplitN(string(gomod), "\n", 2)[0])
			}
			project, _ := os.ReadFile(filepath.Join(dir, "internal", "config", "project.go"))
			if !strings.Contains(string(project), `const Title = "Rocket \"RR\" Rumble"`) {
				t.Error("the title was not set in project.go")
			}
			readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
			if !strings.Contains(string(readme), "Created from [gameskeleton](https://github.com/comprofix/gameskeleton) v"+SkeletonVersion) {
				t.Errorf("README does not say where it came from:\n%s", readme)
			}
			if !keep {
				// A game with no sprites yet can still regenerate its assets.
				cmd := exec.Command("go", "generate", "./assets")
				cmd.Dir = dir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("go generate with no sprites: %v\n%s", err, out)
				}
			}
			if left := leftovers(t, dir); len(left) > 0 {
				t.Errorf("still naming the skeleton:\n%s", strings.Join(left, "\n"))
			}
		})
	}
}

func TestRefuses(t *testing.T) {
	good := options{module: "example.com/me/game", name: "game", title: "Game"}
	for name, o := range map[string]options{
		"no module":     {name: "game", title: "Game"},
		"same module":   {module: oldModule, name: "game", title: "Game"},
		"bad name":      {module: good.module, name: "My Game", title: "Game"},
		"same name":     {module: good.module, name: oldName, title: "Game"},
		"no title":      {module: good.module, name: "game", title: "  "},
		"no skeleton":   good,
		"design exists": good,
	} {
		dir := t.TempDir()
		o.dir = dir
		switch name {
		case "no skeleton":
			os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/other\n"), 0o644)
		case "design exists":
			os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+oldModule+"\n"), 0o644)
			os.WriteFile(filepath.Join(dir, "DESIGN.md"), []byte("# mine\n"), 0o644)
		}
		if err := run(o, &bytes.Buffer{}); err == nil {
			t.Errorf("%s: no error", name)
		}
		if name == "design exists" && !exists(dir, "DESIGN.md") {
			t.Error("removed DESIGN.md")
		}
	}
}
