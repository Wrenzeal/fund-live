package repository

const (
	defaultHoldingTransactionLimit = 20
	maxHoldingTransactionLimit     = 50
)

func normalizeHoldingTransactionLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultHoldingTransactionLimit
	case limit > maxHoldingTransactionLimit:
		return maxHoldingTransactionLimit
	default:
		return limit
	}
}
