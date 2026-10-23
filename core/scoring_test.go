package core

import (
	"math"
	"testing"
	"time"
)

func TestCalculateTextScore(t *testing.T) {
	scorer := NewScorer()
	
	tests := []struct {
		name  string
		text  string
		query string
		want  float64
	}{
		{"exact match", "project", "project", 1.0},
		{"case insensitive", "Project", "project", 1.0},
		{"prefix match", "project-manager", "project", 0.94},
		{"prefix near-full", "project-x", "project", 0.97},
		{"interior word match", "my-project", "project", 0.85},
		{"dense subsequence", "production", "prd", 0.72},
		{"interior token", "my-cool-project", "cool", 0.85},
		{"multi-token (space matches separator)", "react-native-app", "react app", 0.85},
		{"camelCase boundary", "myProject", "project", 0.86},
		// A typo that introduces a character absent from the name is no longer a
		// subsequence, so it scores 0 (the old edit-distance fallback is gone).
		{"typo with missing char", "projet", "project", 0.0},
		{"no match", "something", "xyz", 0.0},
		{"empty query", "anything", "", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scorer.calculateTextScore(tt.text, tt.query)
			tolerance := 0.05
			if math.Abs(got-tt.want) > tolerance {
				t.Errorf("calculateTextScore(%q, %q) = %.3f, want ~%.2f",
					tt.text, tt.query, got, tt.want)
			}
		})
	}
}

// TestTextScoreOrdering pins the relative ordering that motivated the fzf-style
// rewrite. These assertions are independent of the exact scoring constants, so
// they keep protecting intent even if the weights are later retuned.
func TestTextScoreOrdering(t *testing.T) {
	scorer := NewScorer()

	type pair struct {
		query    string
		stronger string // should score strictly higher
		weaker   string
		why      string
	}
	pairs := []pair{
		{"app", "approve-this-very-long-name", "xapp",
			"prefix match must beat an interior match"},
		{"rs", "redis-streams", "rust",
			"hitting both word initials must beat a scattered subsequence"},
		{"rna", "react-native-app", "arena",
			"three word initials must beat a scattered subsequence"},
		{"project", "project-manager", "my-project",
			"prefix must beat the same word appearing mid-name"},
		{"cool", "cool-app", "my-cool-thing",
			"a boundary-anchored match must beat the same word deeper in the name"},
	}

	for _, p := range pairs {
		t.Run(p.query+":"+p.stronger+">"+p.weaker, func(t *testing.T) {
			strong := scorer.calculateTextScore(p.stronger, p.query)
			weak := scorer.calculateTextScore(p.weaker, p.query)
			if !(strong > weak) {
				t.Errorf("query %q: expected %q (%.3f) > %q (%.3f) — %s",
					p.query, p.stronger, strong, p.weaker, weak, p.why)
			}
		})
	}
}

func TestCalculateTimeScore(t *testing.T) {
	scorer := NewScorer()
	scorer.TimeDecayDays = 30 // 30 days for 50% decay
	
	tests := []struct {
		name      string
		daysAgo   float64
		wantScore float64
	}{
		{"today", 0, 1.0},
		{"1 day ago", 1, 0.977},
		{"7 days ago", 7, 0.84},
		{"30 days ago", 30, 0.5},
		{"60 days ago", 60, 0.25},
		{"90 days ago", 90, 0.125},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modTime := time.Now().Add(-time.Duration(tt.daysAgo*24) * time.Hour)
			got := scorer.calculateTimeScore(modTime)
			
			tolerance := 0.05
			if math.Abs(got-tt.wantScore) > tolerance {
				t.Errorf("calculateTimeScore(%v days ago) = %.3f, want ~%.3f", 
					tt.daysAgo, got, tt.wantScore)
			}
		})
	}
}

func TestIsSubsequence(t *testing.T) {
	tests := []struct {
		query string
		text  string
		want  bool
	}{
		{"abc", "aabbcc", true},
		{"prd", "production", true},
		{"xyz", "xaybzc", true},
		{"abc", "abcd", true},
		{"xyz", "abc", false},
		{"", "anything", true},
		{"a", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.query+" in "+tt.text, func(t *testing.T) {
			got := isSubsequence(tt.query, tt.text)
			if got != tt.want {
				t.Errorf("isSubsequence(%q, %q) = %v, want %v", 
					tt.query, tt.text, got, tt.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"my-project", []string{"my", "project"}},
		{"my_cool_project", []string{"my", "cool", "project"}},
		{"MyProject", []string{"myproject", "my", "project"}},
		{"react-native-app", []string{"react", "native", "app"}},
		{"APIClient", []string{"apiclient", "api", "client"}},
		{"simple", []string{"simple"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tokenize(tt.input)
			if !sliceEqual(got, tt.want) {
				t.Errorf("tokenize(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1   string
		s2   string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"project", "projet", 1},
	}
	
	for _, tt := range tests {
		t.Run(tt.s1+" to "+tt.s2, func(t *testing.T) {
			got := levenshteinDistance(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", 
					tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

func TestExtractNameFromDirectory(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2025-08-30-my-project", "my-project"},
		{"2025-01-01-test", "test"},
		{"not-a-date", "not-a-date"},
		{"2025-08-30", "2025-08-30"},
		{"", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractNameFromDirectory(tt.input)
			if got != tt.want {
				t.Errorf("ExtractNameFromDirectory(%q) = %q, want %q", 
					tt.input, got, tt.want)
			}
		})
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}