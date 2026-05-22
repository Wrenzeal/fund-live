'use client'

import { HoldingAggregateRow } from '@/components/holding-aggregate-row'
import { HoldingFundRow } from '@/components/holding-fund-row'
import type { FundAnalysis, FundEstimate } from '@/hooks/use-fund-data'
import type { HoldingAggregateEntry, HoldingEntry, HoldingEstimateAggregateMetrics } from '@/hooks/use-user-portfolio'
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
}: HoldingsListProps) {
  return (
    <div className="space-y-4">
      {viewMode === 'aggregate'
        ? sortedHoldingAggregates.map((aggregate) => (
            <HoldingAggregateRow
              key={aggregate.fund_id}
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
                />
              ))}
            </HoldingAggregateRow>
          ))
        : sortedHoldings.map((holding) => (
            <HoldingFundRow
              key={holding.id}
              holding={holding}
              metricScope={metricScope}
              estimate={estimatesByFundID[holding.fund_id]}
              analysis={analysesByFundID[holding.fund_id]}
              onRemove={() => onRemoveHolding(holding.id)}
            />
          ))}
    </div>
  )
}
