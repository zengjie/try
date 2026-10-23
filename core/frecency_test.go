package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTextWeight(t *testing.T) {
	s := NewScorer()
	cases := []struct {
		qlen int
		want float64
	}{
		{0, 0.0}, {1, 0.45}, {2, 0.45}, {3, 0.65}, {4, 0.65}, {5, 0.80}, {12, 0.80},
	}
	for _, c := range cases {
		if got := s.textWeight(c.qlen); got != c.want {
			t.Errorf("textWeight(%d) = %.2f, want %.2f", c.qlen, got, c.want)
		}
	}
	// Weight must never decrease as the query gets more specific.
	prev := -1.0
	for n := 0; n <= 20; n++ {
		w := s.textWeight(n)
		if w < prev {
			t.Errorf("textWeight not monotonic at %d: %.2f < %.2f", n, w, prev)
		}
		prev = w
	}
}

func TestFrecencyScore(t *testing.T) {
	s := NewScorer()
	now := time.Now()

	if got := s.calculateFrecencyScore(0, now); got != 0 {
		t.Errorf("zero count should score 0, got %.3f", got)
	}

	// At equal frequency, more recent wins.
	recent := s.calculateFrecencyScore(5, now.Add(-1*24*time.Hour))
	stale := s.calculateFrecencyScore(5, now.Add(-60*24*time.Hour))
	if !(recent > stale) {
		t.Errorf("recent (%.3f) should beat stale (%.3f) at equal count", recent, stale)
	}

	// At equal recency, more frequent wins.
	often := s.calculateFrecencyScore(20, now.Add(-7*24*time.Hour))
	rare := s.calculateFrecencyScore(1, now.Add(-7*24*time.Hour))
	if !(often > rare) {
		t.Errorf("frequent (%.3f) should beat rare (%.3f) at equal recency", often, rare)
	}

	for _, v := range []float64{recent, stale, often, rare} {
		if v < 0 || v > 1 {
			t.Errorf("frecency score out of [0,1]: %.3f", v)
		}
	}
}

func TestUsageScoreFallback(t *testing.T) {
	s := NewScorer()
	now := time.Now()
	mod := now.Add(-5 * 24 * time.Hour)

	// With no access history the usage signal is the damped mtime recency.
	noHist := s.calculateUsageScore(mod, 0, time.Time{})
	wantFallback := s.calculateTimeScore(mod) * s.FallbackDamping
	if absdiff(noHist, wantFallback) > 1e-9 {
		t.Errorf("fallback usage = %.4f, want %.4f", noHist, wantFallback)
	}

	// A directory actually opened recently outranks an untracked one of the same age.
	tracked := s.calculateUsageScore(mod, 6, now.Add(-1*24*time.Hour))
	if !(tracked > noHist) {
		t.Errorf("tracked usage (%.3f) should beat untracked (%.3f)", tracked, noHist)
	}
}

func TestFilterAndScoreFrecencyOrdering(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir()) // hermetic: ignore any real user config
	now := time.Now()
	mod := now.Add(-10 * 24 * time.Hour)
	dirs := []Directory{
		{Name: "2025-01-01-rarely", Path: "/x/rarely", ModifiedTime: mod},
		{Name: "2025-01-01-favorite", Path: "/x/favorite", ModifiedTime: mod,
			AccessCount: 8, LastAccess: now.Add(-1 * 24 * time.Hour)},
	}
	scored := FilterAndScoreDirectories(dirs, "")
	SortDirectoriesByScore(scored)
	if len(scored) == 0 || scored[0].Name != "2025-01-01-favorite" {
		t.Errorf("empty query should rank the frequently-used dir first, got %v", names(scored))
	}
}

func TestFilterExcludesNonMatches(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir()) // hermetic: ignore any real user config
	now := time.Now()
	dirs := []Directory{
		{Name: "2025-01-01-redis", Path: "/x/redis", ModifiedTime: now},
		// Heavy usage, but no text match: must still be excluded under a query.
		{Name: "2025-01-01-postgres", Path: "/x/postgres", ModifiedTime: now,
			AccessCount: 50, LastAccess: now},
	}
	scored := FilterAndScoreDirectories(dirs, "redis")
	if len(scored) != 1 || scored[0].Name != "2025-01-01-redis" {
		t.Errorf("query should exclude non-matches even with high usage, got %v", names(scored))
	}
}

func TestShortQueryUsageBreaksWeakTextTie(t *testing.T) {
	t.Setenv("TRY_DATA_DIR", t.TempDir()) // hermetic: ignore any real user config
	now := time.Now()
	old := now.Add(-30 * 24 * time.Hour)
	// Both prefix-match "re". "redis-lab" actually has the slightly higher text
	// score (shorter name), but "redis-cache" is heavily used, so under a short,
	// low-information query usage should float it to the top.
	dirs := []Directory{
		{Name: "2025-01-01-redis-lab", Path: "/x/lab", ModifiedTime: old},
		{Name: "2025-01-01-redis-cache", Path: "/x/cache", ModifiedTime: old,
			AccessCount: 25, LastAccess: now},
	}
	scored := FilterAndScoreDirectories(dirs, "re")
	SortDirectoriesByScore(scored)
	if scored[0].Name != "2025-01-01-redis-cache" {
		t.Errorf("short query should let usage float the used dir up, got %v", names(scored))
	}
}

func TestRecordAndLoadHistory(t *testing.T) {
	tryDir := t.TempDir()
	t.Setenv("TRY_PATH", tryDir)
	t.Setenv("TRY_DATA_DIR", t.TempDir())

	target := filepath.Join(tryDir, "2025-01-01-exp")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := RecordAccess(target); err != nil {
			t.Fatalf("RecordAccess: %v", err)
		}
	}

	rec, ok := LoadHistory()[target]
	if !ok {
		t.Fatalf("history missing the recorded path")
	}
	if rec.Count != 2 {
		t.Errorf("count = %d, want 2", rec.Count)
	}
	if rec.LastAccess.IsZero() {
		t.Errorf("last access not recorded")
	}

	// Paths outside TRY_PATH are not tracked.
	if err := RecordAccess("/somewhere/else"); err != nil {
		t.Fatalf("RecordAccess outside try path: %v", err)
	}
	if _, ok := LoadHistory()["/somewhere/else"]; ok {
		t.Errorf("path outside the try dir should not be tracked")
	}
}

func TestPruneMissingOnRecord(t *testing.T) {
	tryDir := t.TempDir()
	t.Setenv("TRY_PATH", tryDir)
	t.Setenv("TRY_DATA_DIR", t.TempDir())

	gone := filepath.Join(tryDir, "2025-01-01-gone")
	keep := filepath.Join(tryDir, "2025-01-01-keep")
	os.MkdirAll(gone, 0755)
	os.MkdirAll(keep, 0755)

	RecordAccess(gone)
	os.RemoveAll(gone) // directory deleted out from under the history
	RecordAccess(keep) // triggers prune of stale entries

	h := LoadHistory()
	if _, ok := h[gone]; ok {
		t.Errorf("deleted directory should be pruned from history")
	}
	if _, ok := h[keep]; !ok {
		t.Errorf("existing directory should remain in history")
	}
}

func absdiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

func names(dirs []Directory) []string {
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, d.Name)
	}
	return out
}
