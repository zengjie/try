package core

import (
	"math"
	"strings"
	"time"
)

// Score represents a scored directory result
type Score struct {
	Path       string
	Name       string
	Score      float64
	TextScore  float64
	TimeScore  float64
	ModTime    time.Time
}

// Scorer calculates relevance scores for directories
type Scorer struct {
	// TimeDecayDays controls how fast scores decay over time
	// Default is 30 days for 50% decay
	TimeDecayDays float64
}

// NewScorer creates a new scorer with default settings
func NewScorer() *Scorer {
	return &Scorer{
		TimeDecayDays: 30,
	}
}

// ScoreDirectory calculates a relevance score for a directory
func (s *Scorer) ScoreDirectory(dirName string, query string, modTime time.Time) Score {
	// Extract the name part without date prefix
	name := ExtractNameFromDirectory(dirName)
	
	// Calculate text similarity score (0-1)
	textScore := s.calculateTextScore(name, query)
	
	// Calculate time-based score (0-1)
	timeScore := s.calculateTimeScore(modTime)
	
	// Combine scores with weighted average
	// Text match is more important than recency
	finalScore := (textScore * 0.7) + (timeScore * 0.3)
	
	return Score{
		Path:      dirName,
		Name:      name,
		Score:     finalScore,
		TextScore: textScore,
		TimeScore: timeScore,
		ModTime:   modTime,
	}
}

// calculateTextScore computes similarity between query and directory name
func (s *Scorer) calculateTextScore(name, query string) float64 {
	if query == "" {
		return 0.5 // Neutral score for empty query
	}
	
	name = strings.ToLower(name)
	query = strings.ToLower(query)
	
	// Exact match
	if name == query {
		return 1.0
	}
	
	// Prefix match (strong signal)
	if strings.HasPrefix(name, query) {
		// Score based on how much of the name is matched
		return 0.8 + (0.2 * float64(len(query)) / float64(len(name)))
	}
	
	// Contains match
	if strings.Contains(name, query) {
		// Score based on position and length
		position := strings.Index(name, query)
		positionScore := 1.0 - (float64(position) / float64(len(name)))
		lengthScore := float64(len(query)) / float64(len(name))
		return 0.5 + (0.3 * positionScore) + (0.2 * lengthScore)
	}
	
	// Fuzzy match using subsequence matching
	if isSubsequence(query, name) {
		// Calculate density of match
		matchDensity := float64(len(query)) / float64(len(name))
		return 0.3 + (0.4 * matchDensity)
	}
	
	// Token-based matching for multi-word queries
	queryTokens := tokenize(query)
	nameTokens := tokenize(name)
	if len(queryTokens) > 1 || len(nameTokens) > 1 {
		tokenScore := s.calculateTokenScore(queryTokens, nameTokens)
		if tokenScore > 0 {
			return tokenScore * 0.7
		}
	}
	
	// Levenshtein distance for close matches
	distance := levenshteinDistance(query, name)
	maxLen := max(len(query), len(name))
	if distance <= maxLen/3 { // Allow up to 1/3 character differences
		return 0.2 * (1.0 - float64(distance)/float64(maxLen))
	}
	
	return 0.0
}

// calculateTimeScore computes a score based on how recent the directory is
func (s *Scorer) calculateTimeScore(modTime time.Time) float64 {
	daysSince := time.Since(modTime).Hours() / 24
	
	// Use exponential decay
	// Score = e^(-λt) where λ is decay constant
	lambda := math.Ln2 / s.TimeDecayDays // Half-life decay
	score := math.Exp(-lambda * daysSince)
	
	// Ensure score is between 0 and 1
	return math.Max(0, math.Min(1, score))
}

// calculateTokenScore scores based on matching tokens/words
func (s *Scorer) calculateTokenScore(queryTokens, nameTokens []string) float64 {
	if len(queryTokens) == 0 || len(nameTokens) == 0 {
		return 0
	}
	
	matches := 0
	for _, qt := range queryTokens {
		for _, nt := range nameTokens {
			if strings.HasPrefix(nt, qt) {
				matches++
				break
			}
		}
	}
	
	// Score based on percentage of query tokens matched
	return float64(matches) / float64(len(queryTokens))
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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