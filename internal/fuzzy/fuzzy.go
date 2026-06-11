// Package fuzzy provides a tiny, dependency-free subsequence matcher used to
// filter lists by typed input. A query matches a target when all of the query's
// runes appear in the target in order (case-insensitive), so typing "adpg"
// finds "Add Page". It is deliberately simple — no scoring or ranking — because
// the lists it filters are short and preserving the source order (e.g. newest
// PR first) is more useful than re-ranking.
package fuzzy

import "strings"

// Match reports whether query is a case-insensitive subsequence of target.
// An empty query matches everything.
func Match(query, target string) bool {
	if query == "" {
		return true
	}
	q := []rune(strings.ToLower(query))
	i := 0
	for _, r := range strings.ToLower(target) {
		if r == q[i] {
			i++
			if i == len(q) {
				return true
			}
		}
	}
	return false
}
