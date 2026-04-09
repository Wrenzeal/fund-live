package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type TencentMinuteFetcher struct {
	client   *http.Client
	location *time.Location
}

type TencentMinutePoint struct {
	Timestamp time.Time
	Price     decimal.Decimal
	Volume    decimal.Decimal
	Amount    decimal.Decimal
}

type tencentMinuteResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data map[string]struct {
		Data struct {
			Data []string `json:"data"`
			Date string   `json:"date"`
		} `json:"data"`
	} `json:"data"`
}

func NewTencentMinuteFetcher() *TencentMinuteFetcher {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*60*60)
	}

	return &TencentMinuteFetcher{
		client:   &http.Client{Timeout: 20 * time.Second},
		location: loc,
	}
}

func (f *TencentMinuteFetcher) FetchIntradayMinutes(ctx context.Context, code string) ([]TencentMinutePoint, error) {
	symbol := buildTencentMinuteSymbol(code)
	apiURL := fmt.Sprintf("https://ifzq.gtimg.cn/appstock/app/minute/query?code=%s", symbol)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://gu.qq.com/")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tencent minute API returned status %d", resp.StatusCode)
	}

	var parsed tencentMinuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("tencent minute API returned code %d", parsed.Code)
	}

	entry, ok := parsed.Data[symbol]
	if !ok {
		return nil, fmt.Errorf("tencent minute API missing data for %s", symbol)
	}

	tradingDate := strings.TrimSpace(entry.Data.Date)
	if tradingDate == "" {
		return nil, nil
	}

	points := make([]TencentMinutePoint, 0, len(entry.Data.Data))
	for _, line := range entry.Data.Data {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ts, err := time.ParseInLocation("200601021504", tradingDate+fields[0], f.location)
		if err != nil {
			continue
		}

		price, err := decimal.NewFromString(fields[1])
		if err != nil {
			continue
		}

		point := TencentMinutePoint{
			Timestamp: ts,
			Price:     price,
		}
		if len(fields) > 2 {
			point.Volume, _ = decimal.NewFromString(fields[2])
		}
		if len(fields) > 3 {
			point.Amount, _ = decimal.NewFromString(fields[3])
		}
		points = append(points, point)
	}

	return points, nil
}

func buildTencentMinuteSymbol(code string) string {
	if code == "" {
		return code
	}
	if len(code) == 5 {
		return "hk" + code
	}

	firstChar := code[0]
	switch {
	case firstChar == '5' || firstChar == '6' || firstChar == '9':
		return "sh" + code
	case firstChar == '4' || firstChar == '8':
		return "bj" + code
	default:
		return "sz" + code
	}
}
