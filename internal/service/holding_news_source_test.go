package service

import (
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

func TestClassifyAnnouncementTitleRejectsGenericAnnouncement(t *testing.T) {
	impact, strength, accepted := classifyAnnouncementTitle("关于召开股东大会的公告")
	if accepted {
		t.Fatalf("generic announcement should not be accepted, got impact=%s strength=%s", impact, strength)
	}
}

func TestClassifyAnnouncementTitleRecognizesOperationalSignals(t *testing.T) {
	impact, strength, accepted := classifyAnnouncementTitle("关于签署重大订单的进展公告")
	if !accepted {
		t.Fatal("expected operational positive signal to be accepted")
	}
	if impact != "positive" || strength != "high" {
		t.Fatalf("impact=%s strength=%s, want positive/high", impact, strength)
	}
}

func TestEventImpactRankPrefersHeavierHoldingWeight(t *testing.T) {
	heavy := eventImpactRank(domain.FundAnalysisEventImpact{
		Impact:      "positive",
		Strength:    "high",
		TargetScope: "holding",
		Horizon:     "current",
		WeightHint:  decimalPointerFromValue(decimal.RequireFromString("9.5")),
	})
	light := eventImpactRank(domain.FundAnalysisEventImpact{
		Impact:      "positive",
		Strength:    "high",
		TargetScope: "holding",
		Horizon:     "current",
		WeightHint:  decimalPointerFromValue(decimal.RequireFromString("2.0")),
	})

	if heavy <= light {
		t.Fatalf("heavy rank=%d should be greater than light rank=%d", heavy, light)
	}
}
