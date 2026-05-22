package appconfig

import (
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
)

func TestResolveOverseasQuoteSource(t *testing.T) {
	t.Run("defaults to tencent to preserve existing fixed overseas source", func(t *testing.T) {
		t.Setenv(EnvQuoteOverseasSource, "")

		if got := ResolveOverseasQuoteSource(nil); got != domain.QuoteSourceTencent {
			t.Fatalf("ResolveOverseasQuoteSource(nil) = %q, want %q", got, domain.QuoteSourceTencent)
		}
	})

	t.Run("reads config value", func(t *testing.T) {
		t.Setenv(EnvQuoteOverseasSource, "")
		cfg := &Config{
			Quote: QuoteConfig{OverseasSource: string(domain.QuoteSourceSina)},
		}

		if got := ResolveOverseasQuoteSource(cfg); got != domain.QuoteSourceSina {
			t.Fatalf("ResolveOverseasQuoteSource(config) = %q, want %q", got, domain.QuoteSourceSina)
		}
	})

	t.Run("env overrides config", func(t *testing.T) {
		t.Setenv(EnvQuoteOverseasSource, string(domain.QuoteSourceTencent))
		cfg := &Config{
			Quote: QuoteConfig{OverseasSource: string(domain.QuoteSourceSina)},
		}

		if got := ResolveOverseasQuoteSource(cfg); got != domain.QuoteSourceTencent {
			t.Fatalf("ResolveOverseasQuoteSource(env) = %q, want %q", got, domain.QuoteSourceTencent)
		}
	})
}
