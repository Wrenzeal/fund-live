package appconfig

import (
	"os"

	"github.com/RomaticDOG/fund/internal/domain"
)

const EnvQuoteOverseasSource = "QUOTE_OVERSEAS_SOURCE"

func ResolveOverseasQuoteSource(fileCfg *Config) domain.QuoteSource {
	source := domain.QuoteSourceTencent
	if fileCfg != nil {
		source = domain.ResolveQuoteSource(domain.NormalizeQuoteSource(fileCfg.Quote.OverseasSource), source)
	}
	if env := os.Getenv(EnvQuoteOverseasSource); env != "" {
		source = domain.ResolveQuoteSource(domain.NormalizeQuoteSource(env), source)
	}
	return source
}
