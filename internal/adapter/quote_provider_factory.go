package adapter

import "github.com/RomaticDOG/fund/internal/domain"

// NewQuoteProviderForSource builds a quote provider for a configured source.
// Unknown values fall back to Tencent so QDII overseas valuation keeps the
// previous fixed-source behavior unless explicitly configured otherwise.
func NewQuoteProviderForSource(source domain.QuoteSource) domain.QuoteProvider {
	switch domain.ResolveQuoteSource(source, domain.QuoteSourceTencent) {
	case domain.QuoteSourceSina:
		return NewSinaFinanceProvider()
	case domain.QuoteSourceTencent:
		return NewTencentQuoteProvider()
	default:
		return NewTencentQuoteProvider()
	}
}
