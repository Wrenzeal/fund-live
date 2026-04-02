// Package crawler provides data crawling services for fund information.
package crawler

import (
	"context"
	"log"
	"unicode/utf8"

	"github.com/RomaticDOG/fund/internal/adapter"
	"github.com/RomaticDOG/fund/internal/database"
	"gorm.io/gorm"
)

// StockNameFixer provides functionality to fix garbled stock names in the database.
// It uses Sina Finance API to get correct stock names and updates the database.
type StockNameFixer struct {
	db            *gorm.DB
	quoteProvider *adapter.SinaFinanceProvider
}

// NewStockNameFixer creates a new stock name fixer.
func NewStockNameFixer(db *gorm.DB) *StockNameFixer {
	return &StockNameFixer{
		db:            db,
		quoteProvider: adapter.NewSinaFinanceProvider(),
	}
}

// IsGarbled checks if a string contains garbled characters.
// Returns true if the string is not valid UTF-8 or contains replacement characters.
func IsGarbled(s string) bool {
	if s == "" {
		return false
	}

	// Check if valid UTF-8
	if !utf8.ValidString(s) {
		return true
	}

	// Check for common garbled patterns (replacement characters, invalid sequences)
	for _, r := range s {
		// Unicode replacement character
		if r == '\ufffd' {
			return true
		}
		// Check for characters in common garbled ranges
		// These are typical patterns when GBK is misinterpreted
		if r >= 0xE000 && r <= 0xF8FF { // Private Use Area
			return true
		}
	}

	return false
}

// FixGarbledStockNames fixes all garbled stock names in the database.
// It fetches correct names from Sina Finance API and updates the database.
func (f *StockNameFixer) FixGarbledStockNames(ctx context.Context) (int, error) {
	// Find all unique stock codes with potentially garbled names
	var holdings []database.StockHolding
	result := f.db.WithContext(ctx).
		Select("DISTINCT stock_code, stock_name").
		Find(&holdings)

	if result.Error != nil {
		return 0, result.Error
	}

	// Collect stock codes that need fixing
	var codesToFix []string
	codeToOldName := make(map[string]string)

	for _, h := range holdings {
		// Check if name is garbled or missing
		if h.StockName == "" || IsGarbled(h.StockName) {
			codesToFix = append(codesToFix, h.StockCode)
			codeToOldName[h.StockCode] = h.StockName
		}
	}

	if len(codesToFix) == 0 {
		log.Println("✅ No garbled stock names found")
		return 0, nil
	}

	log.Printf("🔍 Found %d stocks with potentially garbled names", len(codesToFix))

	// Fetch correct names from Sina Finance API
	// Process in batches of 50 to avoid overloading the API
	batchSize := 50
	fixedCount := 0

	for i := 0; i < len(codesToFix); i += batchSize {
		end := i + batchSize
		if end > len(codesToFix) {
			end = len(codesToFix)
		}
		batch := codesToFix[i:end]

		quotes, err := f.quoteProvider.GetRealTimeQuotes(ctx, batch)
		if err != nil {
			log.Printf("⚠️ Failed to fetch quotes for batch %d-%d: %v", i, end, err)
			continue
		}

		// Update each stock's name in the database
		for code, quote := range quotes {
			if quote.StockName == "" {
				continue
			}

			oldName := codeToOldName[code]
			newName := quote.StockName

			// Check if the name actually changed (and is valid)
			if newName == oldName || IsGarbled(newName) {
				continue
			}

			// Update all holdings with this stock code
			updateResult := f.db.WithContext(ctx).
				Model(&database.StockHolding{}).
				Where("stock_code = ?", code).
				Update("stock_name", newName)

			if updateResult.Error != nil {
				log.Printf("⚠️ Failed to update stock name for %s: %v", code, updateResult.Error)
				continue
			}

			if updateResult.RowsAffected > 0 {
				log.Printf("   ✅ %s: %q -> %q (%d rows)", code, oldName, newName, updateResult.RowsAffected)
				fixedCount++
			}
		}
	}

	log.Printf("📊 Fixed %d stock names", fixedCount)
	return fixedCount, nil
}

// FixAllStockNames updates ALL stock names (not just garbled ones) from Sina Finance.
// This is useful for a complete refresh of stock names.
func (f *StockNameFixer) FixAllStockNames(ctx context.Context) (int, error) {
	// Get all unique stock codes
	var stockCodes []string
	result := f.db.WithContext(ctx).
		Model(&database.StockHolding{}).
		Distinct("stock_code").
		Pluck("stock_code", &stockCodes)

	if result.Error != nil {
		return 0, result.Error
	}

	if len(stockCodes) == 0 {
		log.Println("✅ No stocks found in database")
		return 0, nil
	}

	log.Printf("🔄 Refreshing names for %d stocks", len(stockCodes))

	// Process in batches
	batchSize := 50
	fixedCount := 0

	for i := 0; i < len(stockCodes); i += batchSize {
		end := i + batchSize
		if end > len(stockCodes) {
			end = len(stockCodes)
		}
		batch := stockCodes[i:end]

		quotes, err := f.quoteProvider.GetRealTimeQuotes(ctx, batch)
		if err != nil {
			log.Printf("⚠️ Failed to fetch quotes for batch %d-%d: %v", i, end, err)
			continue
		}

		for code, quote := range quotes {
			if quote.StockName == "" {
				continue
			}

			updateResult := f.db.WithContext(ctx).
				Model(&database.StockHolding{}).
				Where("stock_code = ?", code).
				Update("stock_name", quote.StockName)

			if updateResult.Error == nil && updateResult.RowsAffected > 0 {
				fixedCount++
			}
		}
	}

	log.Printf("📊 Updated %d stock names", fixedCount)
	return fixedCount, nil
}
