// Package score keeps the high score tables: the best config.TableSize
// scores for each mode, with initials, saved between runs, and which modes
// have been mastered (DESIGN §6.7, D63, D64).
package score

import (
	"encoding/json"
	"os"
	"slices"
	"strings"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Entry is one line of a table. Mastered and Continued mark a run that
// mastered the mode, and the level it reached if it carried on (D63).
type Entry struct {
	Initials  string `json:"initials"`
	Score     int    `json:"score"`
	Mastered  bool   `json:"mastered,omitempty"`
	Continued int    `json:"continued,omitempty"` // the level reached after carrying on; 0 if it didn't
}

// Tables holds a table for each mode, keyed by Mode.ID, best first, and
// which modes have been mastered.
type Tables struct {
	ByMode   map[string][]Entry `json:"tables"`
	Mastered map[string]bool    `json:"mastered,omitempty"`
}

// file is the saved form, which also reads the modes cleared in files
// from before 1.7: a cleared mode counts as mastered (D64).
type file struct {
	Tables
	Cleared map[string]bool `json:"cleared,omitempty"`
}

// New returns empty tables.
func New() *Tables { return &Tables{ByMode: map[string][]Entry{}, Mastered: map[string]bool{}} }

// Load reads the tables at path. A missing or broken file gives empty
// tables, and every entry is checked: tables for unknown modes are
// dropped, initials are cleaned up, scores are clamped, and each table is
// sorted and cut to size. So it never fails.
func Load(path string, modes []config.Mode) *Tables {
	t := New()
	data, err := os.ReadFile(path)
	if err != nil {
		return t
	}
	var f file
	if json.Unmarshal(data, &f) != nil {
		return t
	}
	for _, m := range modes {
		var es []Entry
		for _, e := range f.ByMode[m.ID] {
			if e.Score <= 0 {
				continue
			}
			es = append(es, Entry{Initials: CleanInitials(e.Initials), Score: min(e.Score, config.MaxScore),
				Mastered: e.Mastered, Continued: max(e.Continued, 0)})
		}
		slices.SortStableFunc(es, func(a, b Entry) int { return b.Score - a.Score })
		if len(es) > 0 {
			t.ByMode[m.ID] = es[:min(len(es), config.TableSize)]
		}
		if f.Mastered[m.ID] || f.Cleared[m.ID] {
			t.Mastered[m.ID] = true
		}
	}
	return t
}

// Save writes the tables to path.
func (t *Tables) Save(path string) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteFile(path, append(data, '\n'))
}

// Table is a mode's table, best first.
func (t *Tables) Table(mode string) []Entry { return t.ByMode[mode] }

// Qualifies reports whether score would make a mode's table.
func (t *Tables) Qualifies(mode string, score int) bool {
	es := t.ByMode[mode]
	return score > 0 && (len(es) < config.TableSize || score > es[len(es)-1].Score)
}

// Add puts e in a mode's table and returns its place, 0 for the top, or
// -1 if it didn't make the table. A score equal to one already there goes
// below it: the earlier score keeps its place.
func (t *Tables) Add(mode string, e Entry) int {
	if !t.Qualifies(mode, e.Score) {
		return -1
	}
	e.Score = min(e.Score, config.MaxScore)
	e.Initials = CleanInitials(e.Initials)
	es := t.ByMode[mode]
	i := 0
	for i < len(es) && es[i].Score >= e.Score {
		i++
	}
	es = slices.Insert(es, i, e)
	t.ByMode[mode] = es[:min(len(es), config.TableSize)]
	return i
}

// IsMastered reports whether a mode has been mastered.
func (t *Tables) IsMastered(mode string) bool { return t.Mastered[mode] }

// MarkMastered records that a mode has been mastered, which opens the
// next, and reports whether that is new.
func (t *Tables) MarkMastered(mode string) bool {
	if t.Mastered[mode] {
		return false
	}
	t.Mastered[mode] = true
	return true
}

// Open reports whether mode i of modes can be played: the first always,
// and each later one once the one before it is mastered (D64).
func (t *Tables) Open(modes []config.Mode, i int) bool {
	return i == 0 || t.IsMastered(modes[i-1].ID)
}

// Letters are the characters initials may use.
const Letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// CleanInitials makes s valid initials: upper case, only Letters, padded
// with "A" or cut to config.InitialsLen.
func CleanInitials(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if b.Len() == config.InitialsLen {
			break
		}
		if strings.ContainsRune(Letters, r) {
			b.WriteRune(r)
		}
	}
	for b.Len() < config.InitialsLen {
		b.WriteByte('A')
	}
	return b.String()
}
