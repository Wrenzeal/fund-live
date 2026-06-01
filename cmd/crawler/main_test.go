package main

import (
	"testing"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/domain"
)

func TestResolveFundCatalogStatusMarksBackendShareUnavailable(t *testing.T) {
	tests := []struct {
		name string
		fund crawler.FundListItem
		want string
	}{
		{
			name: "normal catalog fund",
			fund: crawler.FundListItem{Name: "华夏成长混合"},
			want: domain.FundCatalogStatusActive,
		},
		{
			name: "backend share class",
			fund: crawler.FundListItem{Name: "华夏成长混合(后端)"},
			want: domain.FundCatalogStatusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveFundCatalogStatus(tt.fund); got != tt.want {
				t.Fatalf("resolveFundCatalogStatus() = %s, want %s", got, tt.want)
			}
		})
	}
}
