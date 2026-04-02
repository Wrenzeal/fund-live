package crawler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

// EastmoneyHoldingsParser parses fund holdings from Eastmoney HTML.
type EastmoneyHoldingsParser struct{}

// NewEastmoneyHoldingsParser creates a new holdings parser.
func NewEastmoneyHoldingsParser() *EastmoneyHoldingsParser {
	return &EastmoneyHoldingsParser{}
}

// HoldingRawData represents raw holding data from HTML.
type HoldingRawData struct {
	Rank          int    // 排名
	StockCode     string // 股票代码
	StockName     string // 股票名称
	LatestPrice   string // 最新价(可能为空)
	HoldingRatio  string // 持仓占比 (e.g., "8.56%")
	HoldingShares string // 持仓股数(万股)
	MarketValue   string // 持仓市值(万元)
	ReportPeriod  string // 报告期
}

// ParseHoldingsHTML parses the holdings table from Eastmoney FundArchivesDatas HTML.
// The response format: var apidata={ content:"<div class='box'>...</div>", aression:"2024-12-31", ... };
func (p *EastmoneyHoldingsParser) ParseHoldingsHTML(htmlContent string) ([]HoldingRawData, string, error) {
	// Extract the content from the JS response
	htmlTable := p.extractTableHTML(htmlContent)
	if htmlTable == "" {
		// If extraction failed, try using raw content
		htmlTable = htmlContent
	}

	// Extract reporting period
	reportPeriod := p.extractReportPeriod(htmlContent)

	// Parse the HTML table using goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlTable))
	if err != nil {
		return nil, reportPeriod, fmt.Errorf("goquery parse failed: %w", err)
	}

	var holdings []HoldingRawData
	rank := 0

	// The table structure: <table class="w782"><tbody><tr><td>序号</td><td>股票代码</td>...</tr>
	// Find all data rows (skip header)
	doc.Find("table tbody tr").Each(func(i int, row *goquery.Selection) {
		// Skip header row (contains th or first row with specific text)
		if row.Find("th").Length() > 0 {
			return
		}

		cells := row.Find("td")
		cellCount := cells.Length()

		// Need at least 4 cells for meaningful data
		if cellCount < 4 {
			return
		}

		holding := HoldingRawData{}

		// Try to detect table structure by examining cells
		cells.Each(func(j int, cell *goquery.Selection) {
			text := strings.TrimSpace(cell.Text())

			// Look for stock code in links
			cell.Find("a").Each(func(_ int, a *goquery.Selection) {
				href, _ := a.Attr("href")
				// Extract stock code from href like "http://quote.eastmoney.com/sh600519.html"
				codeRe := regexp.MustCompile(`([a-z]{2})(\d{6})\.html`)
				if matches := codeRe.FindStringSubmatch(href); len(matches) > 2 {
					holding.StockCode = matches[2]
				}
				// Also check for code pattern in text
				if holding.StockCode == "" {
					codeTextRe := regexp.MustCompile(`(\d{6})`)
					if matches := codeTextRe.FindStringSubmatch(a.Text()); len(matches) > 1 {
						holding.StockCode = matches[1]
					}
				}
				// Get stock name from link text (second link usually has the name)
				nameText := strings.TrimSpace(a.Text())
				if holding.StockName == "" && nameText != "" && !p.isStockCode(nameText) {
					holding.StockName = nameText
				}
			})

			// Detect column by content pattern
			if p.isStockCode(text) && holding.StockCode == "" {
				holding.StockCode = text
			} else if strings.Contains(text, "%") {
				// This is a ratio column
				if holding.HoldingRatio == "" {
					holding.HoldingRatio = text
				}
			} else if p.isNumeric(text) && len(text) > 0 {
				// Could be shares or market value
				if holding.HoldingShares == "" && j >= 3 {
					holding.HoldingShares = text
				} else if holding.MarketValue == "" && j >= 4 {
					holding.MarketValue = text
				}
			}
		})

		// Additional extraction from td[1] and td[2] for explicit stock code and name
		if cellCount >= 3 {
			// Column 0: Rank
			// Column 1: Stock Code
			if holding.StockCode == "" {
				code := strings.TrimSpace(cells.Eq(1).Text())
				if p.isStockCode(code) {
					holding.StockCode = code
				}
			}
			// Column 2: Stock Name
			if holding.StockName == "" {
				name := strings.TrimSpace(cells.Eq(2).Find("a").Text())
				if name == "" {
					name = strings.TrimSpace(cells.Eq(2).Text())
				}
				holding.StockName = name
			}
			// Column 3: Holding Ratio
			if holding.HoldingRatio == "" {
				holding.HoldingRatio = strings.TrimSpace(cells.Eq(3).Text())
			}
		}

		// Only add if we have valid stock code
		if holding.StockCode != "" {
			rank++
			holding.Rank = rank
			holding.ReportPeriod = reportPeriod
			holdings = append(holdings, holding)
		}
	})

	// If no holdings found via tbody, try direct tr selection
	if len(holdings) == 0 {
		doc.Find("table tr").Each(func(i int, row *goquery.Selection) {
			if row.Find("th").Length() > 0 {
				return
			}

			cells := row.Find("td")
			if cells.Length() < 3 {
				return
			}

			holding := p.parseRowSimple(cells, i+1)
			if holding.StockCode != "" {
				holding.ReportPeriod = reportPeriod
				holdings = append(holdings, holding)
			}
		})
	}

	return holdings, reportPeriod, nil
}

// parseRowSimple provides a simpler row parsing for fallback.
func (p *EastmoneyHoldingsParser) parseRowSimple(cells *goquery.Selection, rank int) HoldingRawData {
	holding := HoldingRawData{Rank: rank}

	// Try to find stock code in any link
	cells.Find("a").Each(func(i int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		text := strings.TrimSpace(a.Text())

		// Extract code from href
		codeRe := regexp.MustCompile(`(\d{6})`)
		if holding.StockCode == "" {
			if matches := codeRe.FindStringSubmatch(href); len(matches) > 1 {
				holding.StockCode = matches[1]
			}
		}

		// Get name from link text
		if holding.StockName == "" && text != "" && !p.isStockCode(text) && len(text) < 20 {
			holding.StockName = text
		}
	})

	// Get ratio from cells with %
	cells.Each(func(i int, cell *goquery.Selection) {
		text := strings.TrimSpace(cell.Text())
		if strings.Contains(text, "%") && holding.HoldingRatio == "" {
			holding.HoldingRatio = text
		}
	})

	return holding
}

// extractTableHTML extracts the HTML table from the JS response.
func (p *EastmoneyHoldingsParser) extractTableHTML(content string) string {
	// Pattern: content:"<div class='box'>...</div>"
	// The JS uses single quotes in attribute values
	contentRe := regexp.MustCompile(`content\s*:\s*"(.*?)",\s*aression`)
	if matches := contentRe.FindStringSubmatch(content); len(matches) > 1 {
		html := matches[1]
		return html
	}

	// Fallback: Try to extract any table content
	tableRe := regexp.MustCompile(`<table[^>]*>[\s\S]*?</table>`)
	if matches := tableRe.FindString(content); matches != "" {
		return matches
	}

	return ""
}

// extractReportPeriod extracts the reporting period from the response.
func (p *EastmoneyHoldingsParser) extractReportPeriod(content string) string {
	// Pattern: aression:"2024-12-31"
	periodRe := regexp.MustCompile(`aression\s*:\s*"(\d{4}-\d{2}-\d{2})"`)
	if matches := periodRe.FindStringSubmatch(content); len(matches) > 1 {
		return matches[1]
	}

	// Try to find date in content
	dateRe := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	if matches := dateRe.FindStringSubmatch(content); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// isStockCode checks if a string looks like a stock code.
func (p *EastmoneyHoldingsParser) isStockCode(s string) bool {
	matched, _ := regexp.MatchString(`^\d{6}$`, s)
	return matched
}

// isNumeric checks if a string is numeric (with possible decimal).
func (p *EastmoneyHoldingsParser) isNumeric(s string) bool {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "%", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	matched, _ := regexp.MatchString(`^-?\d+(\.\d+)?$`, s)
	return matched
}

// ToStockHoldings converts raw holdings data to domain.StockHolding.
func (p *EastmoneyHoldingsParser) ToStockHoldings(rawHoldings []HoldingRawData) []domain.StockHolding {
	result := make([]domain.StockHolding, 0, len(rawHoldings))
	seen := make(map[string]bool) // Deduplicate by stock code

	for _, raw := range rawHoldings {
		// Skip duplicates
		if seen[raw.StockCode] {
			continue
		}
		seen[raw.StockCode] = true

		// Use stock name map as fallback for garbled names
		stockName := GetStockName(raw.StockCode, raw.StockName)

		holding := domain.StockHolding{
			StockCode:       raw.StockCode,
			StockName:       stockName,
			Exchange:        p.inferExchange(raw.StockCode),
			ReportingPeriod: raw.ReportPeriod,
		}

		// Parse holding ratio (remove % sign)
		ratioStr := strings.ReplaceAll(raw.HoldingRatio, "%", "")
		ratioStr = strings.ReplaceAll(ratioStr, ",", "")
		ratioStr = strings.TrimSpace(ratioStr)
		if ratio, err := decimal.NewFromString(ratioStr); err == nil {
			holding.HoldingRatio = ratio
		}

		// Parse holding shares (万股 -> 股)
		sharesStr := strings.ReplaceAll(raw.HoldingShares, ",", "")
		sharesStr = strings.TrimSpace(sharesStr)
		if shares, err := decimal.NewFromString(sharesStr); err == nil {
			holding.HoldingShares = shares.Mul(decimal.NewFromInt(10000))
		}

		// Parse market value (万元 -> 元)
		valueStr := strings.ReplaceAll(raw.MarketValue, ",", "")
		valueStr = strings.TrimSpace(valueStr)
		if value, err := decimal.NewFromString(valueStr); err == nil {
			holding.MarketValue = value.Mul(decimal.NewFromInt(10000))
		}

		result = append(result, holding)
	}

	return result
}

// inferExchange infers the stock exchange from the stock code.
func (p *EastmoneyHoldingsParser) inferExchange(code string) domain.Exchange {
	if len(code) != 6 {
		return domain.ExchangeSH
	}

	prefix := code[:2]
	switch {
	case prefix == "60" || prefix == "68":
		return domain.ExchangeSH
	case prefix == "00" || prefix == "30":
		return domain.ExchangeSZ
	default:
		return domain.ExchangeSH
	}
}
