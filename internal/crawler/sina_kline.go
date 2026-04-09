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

type SinaKLineFetcher struct {
	client   *http.Client
	location *time.Location
}

type SinaKLinePoint struct {
	Timestamp time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
	Amount    decimal.Decimal
}

type sinaKLineResponse struct {
	Result struct {
		Status struct {
			Code int `json:"code"`
		} `json:"status"`
		Data []sinaKLineItem `json:"data"`
	} `json:"result"`
}

type sinaKLineItem struct {
	Day    string `json:"day"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
	Amount string `json:"amount"`
}

func NewSinaKLineFetcher() *SinaKLineFetcher {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*60*60)
	}

	return &SinaKLineFetcher{
		client:   &http.Client{Timeout: 20 * time.Second},
		location: loc,
	}
}

func (f *SinaKLineFetcher) FetchFiveMinuteKLines(ctx context.Context, code string, dataLen int) ([]SinaKLinePoint, error) {
	if dataLen <= 0 {
		dataLen = 242
	}

	symbol := buildSinaKLineSymbol(code)
	apiURL := fmt.Sprintf(
		"https://quotes.sina.cn/cn/api/openapi.php/CN_MarketDataService.getKLineData?symbol=%s&scale=5&ma=no&datalen=%d",
		symbol,
		dataLen,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://finance.sina.com.cn")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sina kline API returned status %d", resp.StatusCode)
	}

	var parsed sinaKLineResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if parsed.Result.Status.Code != 0 {
		return nil, fmt.Errorf("sina kline API returned code %d", parsed.Result.Status.Code)
	}

	points := make([]SinaKLinePoint, 0, len(parsed.Result.Data))
	for _, item := range parsed.Result.Data {
		ts, err := time.ParseInLocation("2006-01-02 15:04:05", item.Day, f.location)
		if err != nil {
			continue
		}

		point := SinaKLinePoint{Timestamp: ts}
		point.Open, _ = decimal.NewFromString(item.Open)
		point.High, _ = decimal.NewFromString(item.High)
		point.Low, _ = decimal.NewFromString(item.Low)
		point.Close, _ = decimal.NewFromString(item.Close)
		point.Volume, _ = decimal.NewFromString(item.Volume)
		point.Amount, _ = decimal.NewFromString(item.Amount)
		points = append(points, point)
	}

	return points, nil
}

func buildSinaKLineSymbol(code string) string {
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

func NormalizeTradingDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func IsTradingDaySeriesPoint(t time.Time, targetDate string) bool {
	return strings.HasPrefix(t.Format("2006-01-02 15:04:05"), targetDate)
}
