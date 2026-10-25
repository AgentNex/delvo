package search

import (
	"strings"

	"github.com/AgentNex/delvo/internal/scanner"
)

// CalculateBonus calculates ranking bonus points for an entry given the query.
func CalculateBonus(e scanner.Entry, query string, caseSensitive bool) int {
	bonus := 0
	name := e.Name
	q := query

	if !caseSensitive {
		name = strings.ToLower(name)
		q = strings.ToLower(q)
	}

	// Exact match bonus
	if name == q {
		bonus += 1000
	} else if strings.HasPrefix(name, q) {
		// Prefix match bonus
		bonus += 500
	} else if strings.Contains(name, q) {
		// Substring match bonus
		bonus += 250
	}

	// Extension match bonus
	if strings.HasPrefix(q, ".") && e.Extension == q {
		bonus += 300
	}

	// Shorter names penalized less / rewarded slightly
	if len(e.Name) < 40 {
		bonus += (40 - len(e.Name)) * 2
	}

	return bonus
}
