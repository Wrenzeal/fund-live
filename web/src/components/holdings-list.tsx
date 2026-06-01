'use client'

import { HoldingAggregateRow } from '@/components/holding-aggregate-row'
import { HoldingFundRow } from '@/components/holding-fund-row'
import { ScrollReveal } from '@/components/scroll-reveal'
import type { FundAnalysis, FundEstimate } from '@/hooks/use-fund-data'
import type {
  AdjustHoldingSharesPayload,
  DividendHoldingPayload,
  HoldingAggregateEntry,
  HoldingEntry,
  HoldingEstimateAggregateMetrics,
  SellHoldingPayload,
  UpdateHoldingPayload,
} from '@/hooks/use-user-portfolio'
import { incompleteAggregateChildren, type HoldingMetricScope } from '@/lib/holding-display'

type HoldingViewMode = 'aggregate' | 'detail'

interface HoldingsListProps {
  viewMode: HoldingViewMode
  metricScope: HoldingMetricScope
  sortedHoldingAggregates: HoldingAggregateEntry[]
  sortedHoldings: HoldingEntry[]
  holdingsByFundID: Record<string, HoldingEntry[]>
  aggregateMetrics: Record<string, HoldingEstimateAggregateMetrics>
  estimatesByFundID: Record<string, FundEstimate | null>
  analysesByFundID: Record<string, FundAnalysis | null>
  showIncompleteOnly: boolean
  onRemoveHolding: (holdingID: string) => void
  onUpdateHolding: (holdingID: string, payload: UpdateHoldingPayload) => Promise<void> | void
  onSellHolding: (holdingID: string, payload: SellHoldingPayload) => Promise<void> | void
  onRecordHoldingDividend: (holdingID: string, payload: DividendHoldingPayload) => Promise<void> | void
  onAdjustHoldingShares: (holdingID: string, payload: AdjustHoldingSharesPayload) => Promise<void> | void
}

export function HoldingsList({
  viewMode,
  metricScope,
  sortedHoldingAggregates,
  sortedHoldings,
  holdingsByFundID,
  aggregateMetrics,
  estimatesByFundID,
  analysesByFundID,
  showIncompleteOnly,
  onRemoveHolding,
  onUpdateHolding,
  onSellHolding,
  onRecordHoldingDividend,
  onAdjustHoldingShares,
}: HoldingsListProps) {
  return (
    <div className="space-y-4">
      {viewMode === 'aggregate'
        ? sortedHoldingAggregates.map((aggregate, index) => (
            <ScrollReveal key={aggregate.fund_id} delay={Math.min(index * 70, 280)}>
              <HoldingAggregateRow
                aggregate={aggregate}
                metricScope={metricScope}
                estimateMetrics={aggregateMetrics[aggregate.fund_id]}
                analysis={analysesByFundID[aggregate.fund_id]}
              >
                {incompleteAggregateChildren(holdingsByFundID[aggregate.fund_id] ?? [], showIncompleteOnly).map((holding) => (
                  <HoldingFundRow
                    key={holding.id}
                    holding={holding}
                    metricScope={metricScope}
                    estimate={estimatesByFundID[holding.fund_id]}
                    analysis={analysesByFundID[holding.fund_id]}
                    onRemove={() => onRemoveHolding(holding.id)}
                    onUpdate={(payload) => onUpdateHolding(holding.id, payload)}
                    onSell={(payload) => onSellHolding(holding.id, payload)}
                    onRecordDividend={(payload) => onRecordHoldingDividend(holding.id, payload)}
                    onAdjustShares={(payload) => onAdjustHoldingShares(holding.id, payload)}
                  />
                ))}
              </HoldingAggregateRow>
            </ScrollReveal>
          ))
        : sortedHoldings.map((holding, index) => (
            <ScrollReveal key={holding.id} delay={Math.min(index * 70, 280)}>
              <HoldingFundRow
                holding={holding}
                metricScope={metricScope}
                estimate={estimatesByFundID[holding.fund_id]}
                analysis={analysesByFundID[holding.fund_id]}
                onRemove={() => onRemoveHolding(holding.id)}
                onUpdate={(payload) => onUpdateHolding(holding.id, payload)}
                onSell={(payload) => onSellHolding(holding.id, payload)}
                onRecordDividend={(payload) => onRecordHoldingDividend(holding.id, payload)}
                onAdjustShares={(payload) => onAdjustHoldingShares(holding.id, payload)}
              />
            </ScrollReveal>
          ))}
    </div>
  )
}
