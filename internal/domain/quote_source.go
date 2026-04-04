package domain

import (
	"context"
	"strings"
)

type QuoteSource string

const (
	QuoteSourceSina    QuoteSource = "sina"
	QuoteSourceTencent QuoteSource = "tencent"
)

type quoteSourceContextKey struct{}

func NormalizeQuoteSource(raw string) QuoteSource {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(QuoteSourceSina):
		return QuoteSourceSina
	case string(QuoteSourceTencent):
		return QuoteSourceTencent
	default:
		return ""
	}
}

func ResolveQuoteSource(preferred, fallback QuoteSource) QuoteSource {
	if normalized := NormalizeQuoteSource(string(preferred)); normalized != "" {
		return normalized
	}
	if normalized := NormalizeQuoteSource(string(fallback)); normalized != "" {
		return normalized
	}
	return QuoteSourceSina
}

func WithQuoteSource(ctx context.Context, source QuoteSource) context.Context {
	return context.WithValue(ctx, quoteSourceContextKey{}, ResolveQuoteSource(source, QuoteSourceSina))
}

func QuoteSourceFromContext(ctx context.Context) QuoteSource {
	if ctx == nil {
		return QuoteSourceSina
	}
	value, ok := ctx.Value(quoteSourceContextKey{}).(QuoteSource)
	if !ok {
		return QuoteSourceSina
	}
	return ResolveQuoteSource(value, QuoteSourceSina)
}
