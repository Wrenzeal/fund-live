'use client'

import useSWR from 'swr'
import { fetchEnvelopeWithTimeout, type Fund } from '@/hooks/use-fund-data'
import { API_BASE_URL } from '@/lib/api-base-url'

export interface FundNotice {
  id: string
  title: string
  scope: 'fund' | 'holding' | 'target_etf'
  published_at: string
  source_name: string
  source_url: string
  related_symbol?: string
  related_name?: string
}

export interface ObservationSource {
  code: 'fund_notices' | 'holding_notices'
  status: 'ok' | 'partial' | 'unavailable' | 'not_applicable'
  checked_at: string
  message?: string
}

export interface FundObservation {
  checked_at: string
  window_days: number
  latest_holding_period: string
  holding_age_days: number | null
  disclosed_weight_percent: string
  holdings_count: number
  holdings_source: string
  holdings_status: 'available' | 'missing' | 'unavailable'
  notices: FundNotice[]
  sources: ObservationSource[]
  notes: string[]
}

interface ObservationPayload {
  fund: Fund
  observation: FundObservation
}

export function useFundObservation(fundId: string) {
  const result = useSWR<{ data: ObservationPayload }, Error>(
    fundId ? `${API_BASE_URL}/api/v1/fund/${encodeURIComponent(fundId)}/observation` : null,
    async (url: string) => {
      const response = await fetchEnvelopeWithTimeout<ObservationPayload>(url, 20_000)
      if (!response.data?.observation || response.data.fund?.id !== fundId) {
        throw new Error('观察资料返回异常，请重新读取')
      }
      return response
    },
    {
      refreshInterval: 600_000,
      dedupingInterval: 30_000,
      revalidateOnFocus: false,
      refreshWhenHidden: false,
      refreshWhenOffline: false,
      keepPreviousData: false,
      shouldRetryOnError: false,
    },
  )
  const payload = result.data?.data
  return { ...result, data: payload?.fund?.id === fundId && payload.observation ? payload : undefined }
}
