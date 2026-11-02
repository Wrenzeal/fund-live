'use client'

import useSWR from 'swr'
import { fetchEnvelopeWithTimeout } from '@/hooks/use-fund-data'
import { API_BASE_URL } from '@/lib/api-base-url'
import type { ObservationPayload } from '@/lib/fund-observation'

export type { FundNotice, ObservationSource, FundObservation, ObservationPayload } from '@/lib/fund-observation'

export function useFundObservation(fundId: string, initialData?: ObservationPayload) {
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
      fallbackData: initialData?.fund.id === fundId ? { data: initialData } : undefined,
    },
  )
  const payload = result.data?.data
  return { ...result, data: payload?.fund?.id === fundId && payload.observation ? payload : undefined }
}
