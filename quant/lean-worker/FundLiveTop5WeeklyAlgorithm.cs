using System;
using System.Collections.Generic;
using System.Globalization;
using System.IO;
using System.Linq;
using Newtonsoft.Json;
using QuantConnect;
using QuantConnect.Algorithm;
using QuantConnect.Algorithm.Framework.Portfolio;
using QuantConnect.Data;
using QuantConnect.Orders;
using QuantConnect.Orders.Fees;
using QuantConnect.Orders.Fills;
using QuantConnect.Orders.Slippage;
using QuantConnect.Securities;

namespace QuantConnect.Algorithm.CSharp
{
    public class FundLiveTop5WeeklyAlgorithm : QCAlgorithm
    {
        private readonly Dictionary<string, Symbol> _marketSymbols = new();
        private readonly Dictionary<string, decimal> _latestScores = new();
        private readonly Dictionary<string, Queue<decimal>> _amountWindows = new();
        private readonly Dictionary<string, Queue<decimal>> _returnWindows = new();
        private readonly Dictionary<string, decimal> _lastAdjustedCloses = new();
        private readonly Dictionary<string, decimal> _latestAdjustedCloses = new();
        private readonly Dictionary<string, int> _observedDays = new();
        private readonly Dictionary<string, decimal> _benchmarkStartPrices = new();
        private readonly Queue<decimal> _benchmarkTrendWindow = new();
        private FundLiveJobManifest _job = new();
        private bool _rebalanceRequested;
        private bool _temporaryRebalanceBuyingPower;

        public override void Initialize()
        {
            var jobDirectory = Environment.GetEnvironmentVariable("FUNDLIVE_LEAN_JOB_DIR");
            if (string.IsNullOrWhiteSpace(jobDirectory))
            {
                throw new InvalidOperationException("FUNDLIVE_LEAN_JOB_DIR is required");
            }

            var manifestPath = Path.Combine(jobDirectory, "job.json");
            _job = JsonConvert.DeserializeObject<FundLiveJobManifest>(File.ReadAllText(manifestPath))
                ?? throw new InvalidOperationException("Unable to decode FundLive job manifest");

            SetTimeZone(TimeZones.Shanghai);
            SetAccountCurrency("CNY");
            SetStartDate(DateTime.ParseExact(_job.Parameters.StartDate, "yyyy-MM-dd", CultureInfo.InvariantCulture));
            SetEndDate(DateTime.ParseExact(_job.Parameters.EndDate, "yyyy-MM-dd", CultureInfo.InvariantCulture));
            SetCash(_job.Parameters.InitialCash);
            foreach (var ticker in _job.Symbols.Concat(new[] { "000300" }).Distinct())
            {
                var properties = new SymbolProperties(ticker, "CNY", 1m, 0.001m, 100m, ticker);
                var exchangeHours = SecurityExchangeHours.AlwaysOpen(TimeZones.Shanghai);
                var security = AddData<FundLiveDailyBar>(ticker, properties, exchangeHours, Resolution.Daily, fillForward: false);
                security.SetFeeModel(new FundLiveEtfFeeModel(_job.Parameters.CommissionBps, _job.Parameters.MinimumCommissionCny));
                security.SetFillModel(new FundLiveNextOpenFillModel());
                security.SetSlippageModel(new FundLiveConstantSlippageModel(_job.Parameters.SlippageBps));
                _marketSymbols[ticker] = security.Symbol;
                _amountWindows[ticker] = new Queue<decimal>();
                _returnWindows[ticker] = new Queue<decimal>();
                _observedDays[ticker] = 0;
            }

            foreach (var ticker in _job.Symbols)
            {
                AddData<FundLiveSignal>(ticker, Resolution.Daily, TimeZones.Shanghai, fillForward: false);
            }

            SetBenchmark(_marketSymbols["000300"]);
            SetWarmUp(Math.Max(_job.Parameters.MinimumListingDays, Math.Max(_job.Parameters.VolatilityLookbackDays, _job.Parameters.TrendFilterDays)), Resolution.Daily);
        }

        public override void OnData(Slice data)
        {
            RestoreCashAccountLeverageWhenSettled();

            foreach (var entry in data.Get<FundLiveDailyBar>())
            {
                var ticker = entry.Key.Value;
                var bar = entry.Value;
                _observedDays[ticker] = _observedDays.GetValueOrDefault(ticker) + 1;
                var window = _amountWindows[ticker];
                window.Enqueue(bar.Amount);
                while (window.Count > 20)
                {
                    window.Dequeue();
                }
                if (bar.AdjustedClose > 0)
                {
                    if (_lastAdjustedCloses.TryGetValue(ticker, out var previous) && previous > 0)
                    {
                        var returns = _returnWindows[ticker];
                        returns.Enqueue(bar.AdjustedClose / previous - 1m);
                        var maximumReturnWindow = Math.Max(20, _job.Parameters.VolatilityLookbackDays);
                        while (returns.Count > maximumReturnWindow)
                        {
                            returns.Dequeue();
                        }
                    }
                    _lastAdjustedCloses[ticker] = bar.AdjustedClose;
                    _latestAdjustedCloses[ticker] = bar.AdjustedClose;
                    if (!_benchmarkStartPrices.ContainsKey(ticker))
                    {
                        _benchmarkStartPrices[ticker] = bar.AdjustedClose;
                    }
                    if (ticker == "000300" && _job.Parameters.TrendFilterDays > 0)
                    {
                        _benchmarkTrendWindow.Enqueue(bar.AdjustedClose);
                        while (_benchmarkTrendWindow.Count > _job.Parameters.TrendFilterDays)
                        {
                            _benchmarkTrendWindow.Dequeue();
                        }
                    }
                }
            }

            foreach (var entry in data.Get<FundLiveSignal>())
            {
                _latestScores[entry.Key.Value] = entry.Value.Score;
                _rebalanceRequested |= entry.Value.IsRebalance;
            }

            PlotBenchmarks();
            if (_rebalanceRequested && !IsWarmingUp)
            {
                RebalancePortfolio();
                _rebalanceRequested = false;
            }
        }

        private void RebalancePortfolio()
        {
            var ranked = _latestScores
                .Where(entry => IsEligible(entry.Key))
                .OrderByDescending(entry => entry.Value)
                .ThenBy(entry => entry.Key, StringComparer.Ordinal)
                .Select((entry, index) => new RankedAsset(entry.Key, entry.Value, index + 1))
                .ToList();

            var selected = ranked
                .Where(entry => entry.Rank <= _job.Parameters.ExitRank && Portfolio[_marketSymbols[entry.Ticker]].Invested)
                .Take(_job.Parameters.TopN)
                .Select(entry => entry.Ticker)
                .ToList();
            foreach (var entry in ranked.Where(entry => entry.Rank <= _job.Parameters.TopN))
            {
                if (selected.Count >= _job.Parameters.TopN)
                {
                    break;
                }
                if (!selected.Contains(entry.Ticker, StringComparer.Ordinal))
                {
                    selected.Add(entry.Ticker);
                }
            }
            var targets = BuildTargetWeights(selected);
            var portfolioTargets = new List<PortfolioTarget>();
            foreach (var ticker in _job.Symbols.OrderBy(value => value, StringComparer.Ordinal))
            {
                var symbol = _marketSymbols[ticker];
                if (!targets.TryGetValue(ticker, out var targetWeight) || targetWeight <= 0m)
                {
                    if (Portfolio[symbol].Invested)
                    {
                        portfolioTargets.Add(new PortfolioTarget(symbol, 0m));
                    }
                    continue;
                }
                var currentWeight = CurrentPortfolioWeight(symbol);
                var differenceBps = Math.Abs(targetWeight - currentWeight) * 10000m;
                if (Portfolio[symbol].Invested && differenceBps < _job.Parameters.MinimumWeightChangeBps)
                {
                    continue;
                }
                portfolioTargets.Add(new PortfolioTarget(symbol, targetWeight));
            }
            if (portfolioTargets.Count > 0)
            {
                var orders = portfolioTargets
                    .Select(target => new RebalanceOrder(target.Symbol, CalculateOrderQuantity(target.Symbol, target.Quantity)))
                    .Where(order => order.Quantity != 0m)
                    .OrderBy(order => order.Quantity > 0m ? 1 : 0)
                    .ThenBy(order => order.Symbol.Value, StringComparer.Ordinal)
                    .ToList();
                if (orders.Count == 0)
                {
                    return;
                }
                foreach (var order in orders)
                {
                    Securities[order.Symbol].SetLeverage(2m);
                }
                _temporaryRebalanceBuyingPower = true;
                foreach (var order in orders)
                {
                    MarketOrder(order.Symbol, order.Quantity, false, $"{_job.Parameters.RebalanceFrequency} ranked rebalance");
                }
            }
        }

        private void RestoreCashAccountLeverageWhenSettled()
        {
            if (!_temporaryRebalanceBuyingPower || Transactions.GetOpenOrders().Count != 0)
            {
                return;
            }
            foreach (var ticker in _job.Symbols)
            {
                Securities[_marketSymbols[ticker]].SetLeverage(1m);
            }
            _temporaryRebalanceBuyingPower = false;
        }

        private Dictionary<string, decimal> BuildTargetWeights(IReadOnlyCollection<string> selected)
        {
            var grossExposure = TargetGrossExposure() * 0.99m;
            if (_job.Parameters.WeightingMethod != "inverse_volatility")
            {
                var equalWeight = grossExposure / _job.Parameters.TopN;
                return selected.ToDictionary(ticker => ticker, _ => equalWeight, StringComparer.Ordinal);
            }

            var inverseVolatility = selected.ToDictionary(
                ticker => ticker,
                ticker => 1m / ReturnVolatility(ticker),
                StringComparer.Ordinal
            );
            var result = new Dictionary<string, decimal>(StringComparer.Ordinal);
            var remaining = selected.ToList();
            var remainingExposure = grossExposure;
            const decimal maximumWeight = 0.30m;
            while (remaining.Count > 0 && remainingExposure > 0)
            {
                var rawTotal = remaining.Sum(ticker => inverseVolatility[ticker]);
                if (rawTotal <= 0)
                {
                    break;
                }
                var newlyCapped = remaining
                    .Where(ticker => remainingExposure * inverseVolatility[ticker] / rawTotal > maximumWeight)
                    .ToList();
                if (newlyCapped.Count == 0)
                {
                    foreach (var ticker in remaining)
                    {
                        result[ticker] = remainingExposure * inverseVolatility[ticker] / rawTotal;
                    }
                    break;
                }
                foreach (var ticker in newlyCapped)
                {
                    result[ticker] = maximumWeight;
                    remainingExposure -= maximumWeight;
                    remaining.Remove(ticker);
                }
            }
            return result;
        }

        private decimal TargetGrossExposure()
        {
            if (_job.Parameters.TrendFilterDays <= 0 || _benchmarkTrendWindow.Count < _job.Parameters.TrendFilterDays)
            {
                return 1m;
            }
            var movingAverage = _benchmarkTrendWindow.Average();
            if (!_latestAdjustedCloses.TryGetValue("000300", out var benchmarkClose) || benchmarkClose >= movingAverage)
            {
                return 1m;
            }
            return _job.Parameters.WeakMarketExposureBps / 10000m;
        }

        private decimal CurrentPortfolioWeight(Symbol symbol)
        {
            if (Portfolio.TotalPortfolioValue <= 0)
            {
                return 0m;
            }
            return Math.Abs(Portfolio[symbol].HoldingsValue) / Portfolio.TotalPortfolioValue;
        }

        private decimal ReturnVolatility(string ticker)
        {
            var values = _returnWindows[ticker].TakeLast(_job.Parameters.VolatilityLookbackDays).Select(value => (double)value).ToArray();
            if (values.Length < _job.Parameters.VolatilityLookbackDays)
            {
                return 0m;
            }
            var mean = values.Average();
            var variance = values.Sum(value => Math.Pow(value - mean, 2)) / (values.Length - 1);
            return (decimal)Math.Sqrt(variance);
        }

        private bool IsEligible(string ticker)
        {
            if (!_marketSymbols.TryGetValue(ticker, out var symbol) || Securities[symbol].Price <= 0)
            {
                return false;
            }
            if (_observedDays.GetValueOrDefault(ticker) < _job.Parameters.MinimumListingDays)
            {
                return false;
            }
            var amounts = _amountWindows[ticker];
            if (amounts.Count != 20 || amounts.Average() < _job.Parameters.MinimumAverageAmount)
            {
                return false;
            }
            return _job.Parameters.WeightingMethod != "inverse_volatility" || ReturnVolatility(ticker) > 0;
        }

        private void PlotBenchmarks()
        {
            if (_benchmarkStartPrices.TryGetValue("000300", out var benchmarkStart) && benchmarkStart > 0 && _latestAdjustedCloses.TryGetValue("000300", out var benchmarkPrice))
            {
                Plot("Benchmarks", "沪深300", benchmarkPrice / benchmarkStart * 100m);
            }

            var poolReturns = new List<decimal>();
            foreach (var ticker in _job.Symbols)
            {
                if (_benchmarkStartPrices.TryGetValue(ticker, out var start) && start > 0 && _latestAdjustedCloses.TryGetValue(ticker, out var current) && current > 0)
                {
                    poolReturns.Add(current / start);
                }
            }
            if (poolReturns.Count > 0)
            {
                Plot("Benchmarks", "试点池等权", poolReturns.Average() * 100m);
            }
            Plot("Benchmarks", "现金", 100m);
        }

        private sealed record RankedAsset(string Ticker, decimal Score, int Rank);
        private sealed record RebalanceOrder(Symbol Symbol, decimal Quantity);
    }

    public class FundLiveDailyBar : BaseData
    {
        public decimal Open { get; set; }
        public decimal High { get; set; }
        public decimal Low { get; set; }
        public decimal Close { get; set; }
        public decimal AdjustedClose { get; set; }
        public decimal Volume { get; set; }
        public decimal Amount { get; set; }
        public decimal AdjustFactor { get; set; }

        public override SubscriptionDataSource GetSource(SubscriptionDataConfig config, DateTime date, bool isLiveMode)
        {
            var root = Environment.GetEnvironmentVariable("FUNDLIVE_LEAN_JOB_DIR") ?? string.Empty;
            return new SubscriptionDataSource(Path.Combine(root, "data", "fundlive", "market", config.Symbol.Value + ".csv"), SubscriptionTransportMedium.LocalFile, FileFormat.Csv);
        }

        public override BaseData Reader(SubscriptionDataConfig config, string line, DateTime date, bool isLiveMode)
        {
            if (string.IsNullOrWhiteSpace(line)) return null;
            var columns = line.Split(',');
            if (columns.Length < 9) return null;
            var time = DateTime.ParseExact(columns[0], "yyyy-MM-dd", CultureInfo.InvariantCulture).AddHours(15);
            var bar = new FundLiveDailyBar
            {
                Symbol = config.Symbol,
                Time = time,
                EndTime = time,
                Open = Parse(columns[1]),
                High = Parse(columns[2]),
                Low = Parse(columns[3]),
                Close = Parse(columns[4]),
                AdjustedClose = Parse(columns[5]),
                Volume = Parse(columns[6]),
                Amount = Parse(columns[7]),
                AdjustFactor = Parse(columns[8])
            };
            bar.Value = bar.Close;
            return bar;
        }

        private static decimal Parse(string value) => decimal.Parse(value, NumberStyles.Any, CultureInfo.InvariantCulture);
    }

    public class FundLiveSignal : BaseData
    {
        public decimal Score { get; set; }
        public decimal ShadowEventScore { get; set; }
        public bool IsRebalance { get; set; }

        public override SubscriptionDataSource GetSource(SubscriptionDataConfig config, DateTime date, bool isLiveMode)
        {
            var root = Environment.GetEnvironmentVariable("FUNDLIVE_LEAN_JOB_DIR") ?? string.Empty;
            return new SubscriptionDataSource(Path.Combine(root, "data", "fundlive", "signals", config.Symbol.Value + ".csv"), SubscriptionTransportMedium.LocalFile, FileFormat.Csv);
        }

        public override BaseData Reader(SubscriptionDataConfig config, string line, DateTime date, bool isLiveMode)
        {
            if (string.IsNullOrWhiteSpace(line)) return null;
            var columns = line.Split(',');
            if (columns.Length < 5) return null;
            var time = DateTime.ParseExact(columns[0], "yyyy-MM-dd", CultureInfo.InvariantCulture).AddHours(15).AddMinutes(1);
            var score = decimal.Parse(columns[1], NumberStyles.Any, CultureInfo.InvariantCulture);
            return new FundLiveSignal
            {
                Symbol = config.Symbol,
                Time = time,
                EndTime = time,
                Score = score,
                ShadowEventScore = decimal.Parse(columns[2], NumberStyles.Any, CultureInfo.InvariantCulture),
                IsRebalance = columns[4] == "1",
                Value = score
            };
        }
    }

    public class FundLiveEtfFeeModel : FeeModel
    {
        private readonly decimal _rate;
        private readonly decimal _minimum;
        public FundLiveEtfFeeModel(decimal basisPoints, decimal minimum)
        {
            _rate = basisPoints / 10000m;
            _minimum = minimum;
        }

        public override OrderFee GetOrderFee(OrderFeeParameters parameters)
        {
            var value = parameters.Security.Price * Math.Abs(parameters.Order.Quantity);
            var fee = Math.Max(_minimum, value * _rate);
            return new OrderFee(new CashAmount(fee, "CNY"));
        }
    }

    public class FundLiveNextOpenFillModel : ImmediateFillModel
    {
        public override OrderEvent MarketFill(Security asset, MarketOrder order)
        {
            var utcTime = asset.LocalTime.ConvertToUtc(asset.Exchange.TimeZone);
            var fill = new OrderEvent(order, utcTime, OrderFee.Zero);
            if (order.Status == OrderStatus.Canceled)
            {
                return fill;
            }

            var bar = asset.GetLastData() as FundLiveDailyBar;
            var localOrderTime = order.Time.ConvertFromUtc(asset.Exchange.TimeZone);
            if (bar == null || bar.EndTime.Date <= localOrderTime.Date || bar.Open <= 0)
            {
                return fill;
            }

            var slippage = asset.SlippageModel.GetSlippageApproximation(asset, order);
            fill.FillPrice = order.Direction == OrderDirection.Buy ? bar.Open + slippage : bar.Open - slippage;
            fill.FillQuantity = order.Quantity;
            fill.Status = OrderStatus.Filled;
            return fill;
        }
    }

    public class FundLiveConstantSlippageModel : ISlippageModel
    {
        private readonly decimal _rate;
        public FundLiveConstantSlippageModel(decimal basisPoints) => _rate = basisPoints / 10000m;
        public decimal GetSlippageApproximation(Security asset, Order order) => asset.Price * _rate;
    }

    public class FundLiveJobManifest
    {
        [JsonProperty("symbols")]
        public List<string> Symbols { get; set; } = new();
        [JsonProperty("parameters")]
        public FundLiveParameters Parameters { get; set; } = new();
    }

    public class FundLiveParameters
    {
        [JsonProperty("start_date")]
        public string StartDate { get; set; } = string.Empty;
        [JsonProperty("end_date")]
        public string EndDate { get; set; } = string.Empty;
        [JsonProperty("initial_cash")]
        public decimal InitialCash { get; set; }
        [JsonProperty("top_n")]
        public int TopN { get; set; }
        [JsonProperty("commission_bps")]
        public decimal CommissionBps { get; set; }
        [JsonProperty("minimum_commission_cny")]
        public decimal MinimumCommissionCny { get; set; }
        [JsonProperty("slippage_bps")]
        public decimal SlippageBps { get; set; }
        [JsonProperty("minimum_listing_days")]
        public int MinimumListingDays { get; set; }
        [JsonProperty("minimum_average_amount")]
        public decimal MinimumAverageAmount { get; set; }
        [JsonProperty("rebalance_frequency")]
        public string RebalanceFrequency { get; set; } = "weekly";
        [JsonProperty("exit_rank")]
        public int ExitRank { get; set; }
        [JsonProperty("minimum_weight_change_bps")]
        public int MinimumWeightChangeBps { get; set; }
        [JsonProperty("weighting_method")]
        public string WeightingMethod { get; set; } = "equal";
        [JsonProperty("volatility_lookback_days")]
        public int VolatilityLookbackDays { get; set; }
        [JsonProperty("trend_filter_days")]
        public int TrendFilterDays { get; set; }
        [JsonProperty("weak_market_exposure_bps")]
        public int WeakMarketExposureBps { get; set; }
    }
}
