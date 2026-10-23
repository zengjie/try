package core

import (
	"math"
	"strings"
	"time"
	"unicode"
)

// Scorer calculates relevance scores for directories. Its fields mirror Config
// (see config.go for what each parameter means and its default).
type Scorer struct {
	TimeDecayDays       float64
	AccessHalfLifeDays  float64
	FrequencySaturation float64
	FallbackDamping     float64
	TextWeightShort     float64
	TextWeightMedium    float64
	TextWeightLong      float64
}

// NewScorer creates a scorer with the built-in defaults. Ranking code paths use
// NewScorerFromConfig(LoadConfig()) instead so user tuning applies; this stays
// deterministic for tests and callers that want the stock behavior.
func NewScorer() *Scorer {
	return NewScorerFromConfig(DefaultConfig())
}

// NewScorerFromConfig creates a scorer using the given (already sanitized)
// ranking parameters.
func NewScorerFromConfig(cfg Config) *Scorer {
	return &Scorer{
		TimeDecayDays:       cfg.TimeDecayDays,
		AccessHalfLifeDays:  cfg.AccessHalfLifeDays,
		FrequencySaturation: cfg.FrequencySaturation,
		FallbackDamping:     cfg.FallbackDamping,
		TextWeightShort:     cfg.TextWeightShort,
		TextWeightMedium:    cfg.TextWeightMedium,
		TextWeightLong:      cfg.TextWeightLong,
	}
}

// Fuzzy scoring constants. A matched character is worth baseScore; it earns
// boundaryBonus when it starts a "word" (name start, after a separator, or a
// camelCase transition) and consecutiveBonus when it directly follows the
// previous matched character. Skipped characters between matches cost
// gapPenalty each. The raw alignment score is normalized against the ideal
// contiguous, boundary-anchored match (baseScore+boundaryBonus per char).
const (
	baseScore        = 1.0
	boundaryBonus    = 1.0
	consecutiveBonus = 1.0
	gapPenalty       = 0.1
	// fuzzyCeiling caps interior fuzzy matches just below the prefix band so an
	// anchored prefix always outranks an equally-clean mid-name match.
	fuzzyCeiling = 0.88
)

// calculateTextScore computes a similarity score in [0,1] between a query and a
// directory name using an fzf-style fuzzy alignment. Unlike a tiered scheme the
// score is continuous and rewards word-boundary hits, consecutive runs, and
// compact matches. This makes acronym-style queries rank intuitively: "rs"
// scores "redis-streams" (both word initials) above "rust" (scattered), and a
// prefix match always beats an equally-dense interior match.
func (s *Scorer) calculateTextScore(name, query string) float64 {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0.5 // Neutral score for empty query
	}

	nameLower := strings.ToLower(name)
	queryLower := strings.ToLower(query)

	// Exact match is the strongest possible signal.
	if nameLower == queryLower {
		return 1.0
	}

	// Prefix match is anchored at the start: ranked just below exact and above
	// any interior match. Greater coverage of the name pushes it toward exact.
	if strings.HasPrefix(nameLower, queryLower) {
		coverage := float64(len([]rune(queryLower))) / float64(len([]rune(nameLower)))
		return 0.90 + 0.09*coverage
	}

	matched, raw, startIdx := fuzzyMatch(name, nameLower, queryLower)
	if !matched {
		return 0.0
	}

	// Normalize against the ideal contiguous, boundary-anchored alignment.
	ideal := (baseScore + boundaryBonus) * float64(len([]rune(queryLower)))
	normalized := raw / ideal
	if normalized > 1.0 {
		normalized = 1.0
	}

	// Cap below the prefix band and break ties toward earlier matches.
	score := fuzzyCeiling*normalized - 0.01*float64(startIdx)
	if score < 0.02 {
		score = 0.02 // any real match stays above the no-match floor of 0
	}
	return score
}

// fuzzyMatch finds the best subsequence alignment of query within name. It
// returns whether query is a subsequence of name, the raw alignment score, and
// the rune index where the best alignment begins. name carries the original
// case (for word-boundary detection); nameLower/queryLower are lowercased.
func fuzzyMatch(name, nameLower, queryLower string) (bool, float64, int) {
	nameRunes := []rune(name)
	nLower := []rune(nameLower)
	qLower := []rune(queryLower)
	ln, lq := len(nLower), len(qLower)
	if lq == 0 || ln < lq {
		return false, 0, 0
	}

	const negInf = -1e9
	// prev[j] is the best raw score aligning query[0..i] with query[i] at
	// name[j]; prevStart[j] is where that alignment began.
	prev := make([]float64, ln)
	prevStart := make([]int, ln)
	for j := range prev {
		prev[j] = negInf
	}

	for i := 0; i < lq; i++ {
		cur := make([]float64, ln)
		curStart := make([]int, ln)
		for j := range cur {
			cur[j] = negInf
		}
		for j := 0; j < ln; j++ {
			if !runesMatch(qLower[i], nLower[j]) {
				continue
			}
			boundary := isBoundaryRune(nameRunes, j)
			if i == 0 {
				v := baseScore
				if boundary {
					v += boundaryBonus
				}
				cur[j] = v
				curStart[j] = j
				continue
			}
			bestVal := negInf
			bestStart := j
			for k := 0; k < j; k++ {
				if prev[k] == negInf {
					continue
				}
				cand := prev[k]
				if gap := j - k - 1; gap > 0 {
					cand -= gapPenalty * float64(gap)
				}
				v := baseScore
				if k == j-1 {
					v += consecutiveBonus
				}
				if boundary {
					v += boundaryBonus
				}
				if cand+v > bestVal {
					bestVal = cand + v
					bestStart = prevStart[k]
				}
			}
			cur[j] = bestVal
			curStart[j] = bestStart
		}
		prev = cur
		prevStart = curStart
	}

	best := negInf
	bestStart := 0
	for j := 0; j < ln; j++ {
		if prev[j] > best {
			best = prev[j]
			bestStart = prevStart[j]
		}
	}
	if best <= negInf/2 {
		return false, 0, 0
	}
	return true, best, bestStart
}

// runesMatch reports whether a query rune matches a name rune. Separators are
// interchangeable so a space in the query matches a '-'/'_' in the name, which
// lets multi-word queries like "react app" match "react-native-app".
func runesMatch(q, n rune) bool {
	return q == n || (isSeparatorRune(q) && isSeparatorRune(n))
}

// isBoundaryRune reports whether name[j] starts a new word: the first rune, a
// rune after a separator, a camelCase transition, or a letter→digit transition.
func isBoundaryRune(name []rune, j int) bool {
	if j == 0 {
		return true
	}
	prev, cur := name[j-1], name[j]
	if isSeparatorRune(prev) {
		return true
	}
	if !unicode.IsUpper(prev) && unicode.IsUpper(cur) {
		return true
	}
	if unicode.IsLetter(prev) && unicode.IsDigit(cur) {
		return true
	}
	return false
}

func isSeparatorRune(r rune) bool {
	switch r {
	case '-', '_', ' ', '.', '/':
		return true
	}
	return false
}

// recencyScore returns an exponential-decay score in [0,1] for a timestamp,
// reaching 0.5 after halfLifeDays. A zero time or non-positive half-life is 0.
func (s *Scorer) recencyScore(t time.Time, halfLifeDays float64) float64 {
	if t.IsZero() || halfLifeDays <= 0 {
		return 0
	}
	daysSince := time.Since(t).Hours() / 24
	lambda := math.Ln2 / halfLifeDays // half-life decay: score = e^(-λt)
	return math.Max(0, math.Min(1, math.Exp(-lambda*daysSince)))
}

// calculateTimeScore computes a score based on how recently the directory was
// modified on disk.
func (s *Scorer) calculateTimeScore(modTime time.Time) float64 {
	return s.recencyScore(modTime, s.TimeDecayDays)
}

// calculateFrecencyScore blends how often (frequency) and how recently (recency)
// a directory was opened through try into a single [0,1] score. Recency leads
// so the thing you touched last floats up, while frequency gives a durable boost
// to directories you return to again and again.
func (s *Scorer) calculateFrecencyScore(count int, lastAccess time.Time) float64 {
	if count <= 0 {
		return 0
	}
	recency := s.recencyScore(lastAccess, s.AccessHalfLifeDays)
	// Saturating frequency: count/(count+K). With K=3, 1 visit -> 0.25,
	// 3 -> 0.5, 9 -> 0.75, so repeated use keeps paying off with diminishing return.
	frequency := float64(count) / (float64(count) + s.FrequencySaturation)
	return clamp01(0.65*recency + 0.35*frequency)
}

// calculateUsageScore is the recency/usage signal used for ranking. It uses
// frecency when the directory has been opened through try before, and otherwise
// falls back to a damped directory-mtime recency so freshly created or cloned
// directories (no history yet) still rank sensibly.
func (s *Scorer) calculateUsageScore(modTime time.Time, count int, lastAccess time.Time) float64 {
	if count > 0 && !lastAccess.IsZero() {
		return s.calculateFrecencyScore(count, lastAccess)
	}
	return s.calculateTimeScore(modTime) * s.FallbackDamping
}

// textWeight returns how much the text-match signal counts relative to the
// usage signal, as a function of query length. A short query carries little
// information so usage dominates; a longer, more specific query trusts the text
// match. An empty query is pure usage ranking.
func (s *Scorer) textWeight(queryLen int) float64 {
	switch {
	case queryLen <= 0:
		return 0.0
	case queryLen <= 2:
		return s.TextWeightShort
	case queryLen <= 4:
		return s.TextWeightMedium
	default:
		return s.TextWeightLong
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// isSubsequence checks if query is a subsequence of text
func isSubsequence(query, text string) bool {
	if len(query) == 0 {
		return true
	}
	if len(text) == 0 {
		return false
	}
	
	queryIdx := 0
	for i := 0; i < len(text) && queryIdx < len(query); i++ {
		if text[i] == query[queryIdx] {
			queryIdx++
		}
	}
	
	return queryIdx == len(query)
}

// tokenize splits text into tokens for matching
func tokenize(text string) []string {
	var tokens []string
	
	// Split by common separators
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})
	
	for _, part := range parts {
		if part != "" {
			tokens = append(tokens, strings.ToLower(part))
		}
	}
	
	// Also add camelCase splits
	for _, part := range parts {
		camelTokens := splitCamelCase(part)
		for _, token := range camelTokens {
			if token != "" && !contains(tokens, strings.ToLower(token)) {
				tokens = append(tokens, strings.ToLower(token))
			}
		}
	}
	
	return tokens
}

// splitCamelCase splits a camelCase string into words
func splitCamelCase(s string) []string {
	var result []string
	var current []rune
	
	for i, r := range s {
		if i > 0 && isUpper(r) && (i+1 < len(s) && !isUpper(rune(s[i+1])) || len(current) > 0 && !isUpper(current[len(current)-1])) {
			if len(current) > 0 {
				result = append(result, string(current))
				current = []rune{}
			}
		}
		current = append(current, r)
	}
	
	if len(current) > 0 {
		result = append(result, string(current))
	}
	
	return result
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}
	
	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}
	
	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}
	
	// Calculate distances
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}
	
	return matrix[len(s1)][len(s2)]
}

// Helper functions
func min(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	minVal := nums[0]
	for _, n := range nums[1:] {
		if n < minVal {
			minVal = n
		}
	}
	return minVal
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ExtractNameFromDirectory removes the date prefix from a directory name
func ExtractNameFromDirectory(dirName string) string {
	// Remove date prefix (YYYY-MM-DD-)
	if len(dirName) > 11 && dirName[4] == '-' && dirName[7] == '-' && dirName[10] == '-' {
		return dirName[11:]
	}
	return dirName
}