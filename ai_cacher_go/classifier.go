// Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

package ai_cacher_go

import "strings"

type ComplexityClassifier struct {
}

func NewComplexityClassifier() *ComplexityClassifier {
	return &ComplexityClassifier{}
}

func (c *ComplexityClassifier) IsLocalTask(userText string, systemPrompt string) bool {
	combined := strings.ToLower(strings.TrimSpace(systemPrompt + " " + userText))
	localKeywords := []string{
		"format", "reformat", "json", "csv", "yaml", "markdown", "summarize",
		"extract", "clean", "normalize", "validate", "transform", "tokenize",
	}
	cloudKeywords := []string{
		"javascript", "python", "sql", "code", "complex", "reason", "analysis",
		"multi-step", "strategy", "plan", "problem", "math", "proof", "logic",
	}

	localScore := 0
	for _, keyword := range localKeywords {
		if strings.Contains(combined, keyword) {
			localScore++
		}
	}

	cloudScore := 0
	for _, keyword := range cloudKeywords {
		if strings.Contains(combined, keyword) {
			cloudScore++
		}
	}

	if localScore >= 2 && cloudScore == 0 {
		return true
	}
	if cloudScore >= 2 && localScore == 0 {
		return false
	}
	if strings.Contains(combined, "simple") || strings.Contains(combined, "basic") {
		return true
	}

	return localScore >= cloudScore
}
