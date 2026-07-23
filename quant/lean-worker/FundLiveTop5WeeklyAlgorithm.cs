using System;
using System.Collections.Generic;
using System.Globalization;
using System.IO;
using System.Linq;
using Newtonsoft.Json;
using QuantConnect;
using QuantConnect.Algorithm;
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
        private readonly Dictionary<string, int> _observedDays = new();
        private readonly Dictionary<string, decimal> _benchmarkStartPrices = new();
        private FundLiveJobManifest _job = new();
        private bool _rebalanceRequested;

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
                var security = AddData<FundLiveDailyBar>(ticker, properties, exchangeHours, Resolution.Daily);
                security.SetFeeModel(new FundLiveEtfFeeModel(_job.Parameters.CommissionBps, _job.Parameters.MinimumCommissionCny));
                security.SetFillModel(new FundLiveNextOpenFillModel());
                security.SetSlippageModel(new FundLiveConstantSlippageModel(_job.Parameters.SlippageBps));
                _marketSymbols[ticker] = security.Symbol;
                _amountWindows[ticker] = new Queue<decimal>();
                _observedDays[ticker] = 0;
            }

            foreach (var ticker in _job.Symbols)
            {
                AddData<FundLiveSignal>(ticker, Resolution.Daily, TimeZones.Shanghai);
            }

            SetBenchmark(_marketSymbols["000300"]);
            SetWarmUp(_job.Parameters.MinimumListingDays, Resolution.Daily);
        }

        public override void OnData(Slice data)
        {
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
                if (!_benchmarkStartPrices.ContainsKey(ticker) && bar.Close > 0)
                {
                    _benchmarkStartPrices[ticker] = bar.Close;
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
            var selected = _latestScores
                .Where(entry => IsEligible(entry.Key))
                .OrderByDescending(entry => entry.Value)
                .ThenBy(entry => entry.Key, StringComparer.Ordinal)
                .Take(_job.Parameters.TopN)
                .Select(entry => entry.Key)
                .ToHashSet(StringComparer.Ordinal);

            foreach (var ticker in _job.Symbols)
            {
                var symbol = _marketSymbols[ticker];
                if (Portfolio[symbol].Invested && !selected.Contains(ticker))
                {
                    Liquidate(symbol, "Weekly Top-N exit");
                }
            }

            if (selected.Count == 0)
            {
                return;
            }
            var targetWeight = 1m / _job.Parameters.TopN;
            foreach (var ticker in selected)
            {
                SetHoldings(_marketSymbols[ticker], targetWeight, liquidateExistingHoldings: false, tag: $"Weekly score {_latestScores[ticker]:F2}");
            }
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
            return amounts.Count == 20 && amounts.Average() >= _job.Parameters.MinimumAverageAmount;
        }

        private void PlotBenchmarks()
        {
            if (_marketSymbols.TryGetValue("000300", out var benchmark) && _benchmarkStartPrices.TryGetValue("000300", out var benchmarkStart) && benchmarkStart > 0)
            {
                Plot("Benchmarks", "沪深300", Securities[benchmark].Price / benchmarkStart * 100m);
            }

            var poolReturns = new List<decimal>();
            foreach (var ticker in _job.Symbols)
            {
                if (_benchmarkStartPrices.TryGetValue(ticker, out var start) && start > 0 && Securities[_marketSymbols[ticker]].Price > 0)
                {
                    poolReturns.Add(Securities[_marketSymbols[ticker]].Price / start);
                }
            }
            if (poolReturns.Count > 0)
            {
                Plot("Benchmarks", "试点池等权", poolReturns.Average() * 100m);
            }
            Plot("Benchmarks", "现金", 100m);
        }
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
    }
}
