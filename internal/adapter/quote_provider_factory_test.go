package adapter

import (
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
)

func TestNewQuoteProviderForSource(t *testing.T) {
	tests := []struct {
		name     string
		source   domain.QuoteSource
		wantName string
	}{
		{
			name:     "sina source",
			source:   domain.QuoteSourceSina,
			wantName: "SinaFinance",
		},
		{
			name:     "tencent source",
			source:   domain.QuoteSourceTencent,
			wantName: "tencent",
		},
		{
			name:     "unknown source falls back to tencent",
			source:   domain.QuoteSource("unknown"),
			wantName: "tencent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewQuoteProviderForSource(tt.source)
			if provider == nil {
				t.Fatal("provider is nil")
			}
			if got := provider.GetName(); got != tt.wantName {
				t.Fatalf("provider.GetName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}
