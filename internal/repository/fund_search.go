package repository

import (
	"sort"
	"strings"

	"github.com/RomaticDOG/fund/internal/domain"
)

const (
	searchCandidateMultiplier = 5
	searchCandidateFloor      = 50
)

type fundSearchMatch struct {
	exactID      bool
	prefixID     bool
	containsID   bool
	prefixName   bool
	containsName bool
	containsMgr  bool
}

func normalizeSearchQuery(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func searchCandidateLimit(limit int) int {
	if limit <= 0 {
		return 0
	}

	candidateLimit := limit * searchCandidateMultiplier
	if candidateLimit < searchCandidateFloor {
		candidateLimit = searchCandidateFloor
	}
	return candidateLimit
}

func classifyFundSearchMatch(fund *domain.Fund, normalizedQuery string) (fundSearchMatch, bool) {
	if fund == nil || normalizedQuery == "" {
		return fundSearchMatch{}, false
	}

	id := strings.ToLower(strings.TrimSpace(fund.ID))
	name := strings.ToLower(strings.TrimSpace(fund.Name))
	manager := strings.ToLower(strings.TrimSpace(fund.Manager))

	match := fundSearchMatch{
		exactID:      id == normalizedQuery,
		prefixID:     strings.HasPrefix(id, normalizedQuery),
		containsID:   strings.Contains(id, normalizedQuery),
		prefixName:   strings.HasPrefix(name, normalizedQuery),
		containsName: strings.Contains(name, normalizedQuery),
		containsMgr:  strings.Contains(manager, normalizedQuery),
	}

	matched := match.exactID || match.prefixID || match.containsID || match.prefixName || match.containsName || match.containsMgr
	return match, matched
}

func betterFundSearchMatch(left, right fundSearchMatch) bool {
	switch {
	case left.exactID != right.exactID:
		return left.exactID
	case left.prefixID != right.prefixID:
		return left.prefixID
	case left.containsID != right.containsID:
		return left.containsID
	case left.prefixName != right.prefixName:
		return left.prefixName
	case left.containsName != right.containsName:
		return left.containsName
	case left.containsMgr != right.containsMgr:
		return left.containsMgr
	default:
		return false
	}
}

func rankAndLimitFunds(candidates []*domain.Fund, query string, limit int) []*domain.Fund {
	normalizedQuery := normalizeSearchQuery(query)
	if normalizedQuery == "" || limit <= 0 || len(candidates) == 0 {
		return []*domain.Fund{}
	}

	type scoredFund struct {
		fund  *domain.Fund
		match fundSearchMatch
	}

	byID := make(map[string]scoredFund, len(candidates))
	for _, fund := range candidates {
		match, ok := classifyFundSearchMatch(fund, normalizedQuery)
		if !ok {
			continue
		}

		existing, exists := byID[fund.ID]
		if !exists || betterFundSearchMatch(match, existing.match) {
			byID[fund.ID] = scoredFund{fund: fund, match: match}
		}
	}

	scored := make([]scoredFund, 0, len(byID))
	for _, item := range byID {
		scored = append(scored, item)
	}

	sort.Slice(scored, func(i, j int) bool {
		left := scored[i]
		right := scored[j]

		if betterFundSearchMatch(left.match, right.match) {
			return true
		}
		if betterFundSearchMatch(right.match, left.match) {
			return false
		}

		leftID := strings.TrimSpace(left.fund.ID)
		rightID := strings.TrimSpace(right.fund.ID)
		if leftID != rightID {
			return leftID < rightID
		}

		leftName := strings.TrimSpace(left.fund.Name)
		rightName := strings.TrimSpace(right.fund.Name)
		return leftName < rightName
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	results := make([]*domain.Fund, 0, len(scored))
	for _, item := range scored {
		results = append(results, item.fund)
	}
	return results
}
