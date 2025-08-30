package core

import (
	"math"
	"strings"
	"time"
	"unicode"
)

func ScoreDirectories(directories []Directory, query string) {
	now := time.Now()
	
	for i := range directories {
		textScore := 0.0
		if query != "" {
			textScore = calculateTextScore(directories[i].Name, query)
		}
		
		timeScore := calculateTimeScore(directories[i].ModifiedTime, now)
		
		if query != "" {
			directories[i].Score = textScore * 0.7 + timeScore * 0.3
		} else {
			directories[i].Score = timeScore
		}
		
		if strings.HasPrefix(directories[i].Name, time.Now().Format("2006-01")) {
			directories[i].Score += 10.0
		}
	}
}

func calculateTextScore(text, query string) float64 {
	text = strings.ToLower(text)
	query = strings.ToLower(query)
	
	if text == query {
		return 1000.0
	}
	
	if strings.HasPrefix(text, query) {
		return 500.0
	}
	
	if !strings.Contains(text, query) {
		return fuzzyMatch(text, query)
	}
	
	index := strings.Index(text, query)
	score := 100.0
	
	if index == 0 || (index > 0 && !unicode.IsLetter(rune(text[index-1]))) {
		score += 50.0
	}
	
	score += float64(100 - index)
	
	score -= float64(len(text)) * 0.5
	
	return score
}

func fuzzyMatch(text, query string) float64 {
	if query == "" {
		return 0
	}
	
	textRunes := []rune(text)
	queryRunes := []rune(query)
	
	score := 0.0
	textIndex := 0
	lastMatchIndex := -1
	consecutiveMatches := 0
	
	for _, queryChar := range queryRunes {
		found := false
		for i := textIndex; i < len(textRunes); i++ {
			if unicode.ToLower(textRunes[i]) == unicode.ToLower(queryChar) {
				found = true
				score += 10.0
				
				if i == 0 || !unicode.IsLetter(textRunes[i-1]) {
					score += 15.0
				}
				
				if lastMatchIndex >= 0 && i == lastMatchIndex+1 {
					consecutiveMatches++
					score += float64(consecutiveMatches) * 5.0
				} else {
					consecutiveMatches = 0
				}
				
				if lastMatchIndex >= 0 {
					gap := i - lastMatchIndex - 1
					score -= float64(gap) * 0.5
				}
				
				lastMatchIndex = i
				textIndex = i + 1
				break
			}
		}
		
		if !found {
			return 0
		}
	}
	
	matchDensity := float64(len(queryRunes)) / float64(lastMatchIndex+1)
	score *= (1.0 + matchDensity)
	
	score -= float64(len(textRunes)) * 0.1
	
	return score
}

func calculateTimeScore(modTime time.Time, now time.Time) float64 {
	hoursSince := now.Sub(modTime).Hours()
	
	if hoursSince < 1 {
		return 100.0
	} else if hoursSince < 24 {
		return 90.0 - (hoursSince * 2)
	} else if hoursSince < 24*7 {
		daysSince := hoursSince / 24
		return 70.0 - (daysSince * 5)
	} else if hoursSince < 24*30 {
		weeksSince := hoursSince / (24 * 7)
		return 40.0 - (weeksSince * 2)
	}
	
	monthsSince := hoursSince / (24 * 30)
	return math.Max(1.0, 20.0-math.Sqrt(monthsSince)*2)
}