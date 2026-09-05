package soul

import (
	"regexp"
	"strings"
)

var phraseHits = []string{
	"ignore previous instructions",
	"ignore all previous",
	"you are now",
	"exfiltrate",
	"drop your system prompt",
}

var apiKeyAdjacent = regexp.MustCompile(`(?i)(api[ _-]?key).{0,24}(print|cat|send)|(print|cat|send).{0,24}(api[ _-]?key)`)

// Scan is warn-only. It never blocks spawn.
func Scan(body string) []string {
	lower := strings.ToLower(body)
	var hits []string
	for _, p := range phraseHits {
		if strings.Contains(lower, p) {
			hits = append(hits, p)
		}
	}
	if apiKeyAdjacent.MatchString(body) {
		hits = append(hits, "api key adjacent to print/cat/send")
	}
	return hits
}
