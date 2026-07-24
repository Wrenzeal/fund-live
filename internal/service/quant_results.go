package service

type QuantSeriesPoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

type QuantBacktestSeries struct {
	Strategy         []QuantSeriesPoint `json:"strategy"`
	CSI300           []QuantSeriesPoint `json:"csi300"`
	PilotEqualWeight []QuantSeriesPoint `json:"pilot_equal_weight"`
	Cash             []QuantSeriesPoint `json:"cash"`
	Drawdown         []QuantSeriesPoint `json:"drawdown"`
}

type QuantBacktestAnnualReturn struct {
	Year                int     `json:"year"`
	TradingDays         int     `json:"trading_days"`
	StrategyReturnPct   float64 `json:"strategy_return_pct"`
	CSI300ReturnPct     float64 `json:"csi300_return_pct"`
	PilotEqualReturnPct float64 `json:"pilot_equal_return_pct"`
	ExcessOverCSI300Pct float64 `json:"excess_over_csi300_pct"`
}

type QuantBacktestMetricSummary struct {
	TotalReturnPct       float64 `json:"total_return_pct"`
	CAGRPct              float64 `json:"cagr_pct"`
	MaxDrawdownPct       float64 `json:"max_drawdown_pct"`
	Sharpe               float64 `json:"sharpe"`
	Sortino              float64 `json:"sortino"`
	InformationRatio     float64 `json:"information_ratio"`
	PortfolioTurnoverPct float64 `json:"portfolio_turnover_pct"`
	TotalFeesCNY         float64 `json:"total_fees_cny"`
	TotalOrders          int     `json:"total_orders"`
	StartEquityCNY       float64 `json:"start_equity_cny"`
	EndEquityCNY         float64 `json:"end_equity_cny"`
	CSI300ReturnPct      float64 `json:"csi300_return_pct"`
	PilotEqualReturnPct  float64 `json:"pilot_equal_return_pct"`
	CashReturnPct        float64 `json:"cash_return_pct"`
	ExcessReturnPct      float64 `json:"excess_return_pct"`
}

type QuantNormalizedResult struct {
	SchemaVersion string                      `json:"schema_version"`
	Summary       QuantBacktestMetricSummary  `json:"summary"`
	Series        QuantBacktestSeries         `json:"series"`
	AnnualReturns []QuantBacktestAnnualReturn `json:"annual_returns"`
}
