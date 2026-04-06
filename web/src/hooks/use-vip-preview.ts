'use client'

import useSWR from 'swr'
import { useCurrentUser } from '@/hooks/use-auth'
import {
  VIP_DAILY_QUOTA,
  VIP_PLAN,
  VIP_SAMPLE_REPORTS,
  type VIPBillingCycle,
  type VIPMembershipState,
  type VIPReport,
  type VIPReportSource,
  type VIPTaskType,
  type VIPTargetType,
  type VIPTaskView,
  getVIPSampleReportByID,
} from '@/mocks/vip'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  error?: {
    code?: string
    message?: string
  }
}

interface RawMembershipState {
  is_vip: boolean
  plan_code: 'vip'
  plan_name: string
  billing_cycle: VIPBillingCycle
  activated_at: string
  expires_at: string
}

interface RawQuotaStatus {
  usage_date: string
  sector_analysis_limit: number
  sector_analysis_used: number
  sector_analysis_remaining: number
  portfolio_analysis_limit: number
  portfolio_analysis_used: number
  portfolio_analysis_remaining: number
}

interface RawTaskView {
  id: string
  type: VIPTaskType
  target_type: VIPTargetType
  target_id: string
  target_name: string
  created_at: string
  status: VIPTaskView['status']
  started_at?: string
  completed_at?: string
  progress_text: string
  report_id?: string
}

interface RawReportSource extends Omit<VIPReportSource, 'publishedAt'> {
  published_at: string
}

interface RawReport extends Omit<VIPReport, 'targetName' | 'generatedAt' | 'coverageWindow' | 'footerDisclaimer' | 'sources'> {
  target_name: string
  generated_at: string
  coverage_window: string
  footerDisclaimer: string
  sources: RawReportSource[]
}

export interface VIPOrder {
  id: string
  orderNo: string
  planCode: 'vip'
  planName: string
  billingCycle: VIPBillingCycle
  amountFen: number
  currency: string
  status: 'pending_payment' | 'paid' | 'closed' | 'failed'
  paymentChannel: 'wechat_pay'
  paymentScene: 'native'
  description: string
  codeURL?: string
  wechatTransactionID?: string
  errorCode?: string
  errorMessage?: string
  expiresAt?: string
  paidAt?: string
  createdAt: string
  updatedAt: string
}

interface RawOrder {
  id: string
  order_no: string
  plan_code: 'vip'
  plan_name: string
  billing_cycle: VIPBillingCycle
  amount_fen: number
  currency: string
  status: VIPOrder['status']
  payment_channel: 'wechat_pay'
  payment_scene: 'native'
  description: string
  code_url?: string
  wechat_transaction_id?: string
  error_code?: string
  error_message?: string
  expires_at?: string
  paid_at?: string
  created_at: string
  updated_at: string
}

export class VIPRequestError extends Error {
  code: string

  constructor(message: string, code?: string) {
    super(message)
    this.name = 'VIPRequestError'
    this.code = code || 'VIP_REQUEST_FAILED'
  }
}

const defaultMembershipState: VIPMembershipState = {
  isVip: false,
  planCode: VIP_PLAN.code,
  planName: VIP_PLAN.name,
  billingCycle: 'monthly',
  activatedAt: '',
  expiresAt: '',
  usageByDate: {},
}

function normalizeMembership(raw?: RawMembershipState | null): VIPMembershipState {
  if (!raw) {
    return defaultMembershipState
  }

  return {
    isVip: raw.is_vip,
    planCode: raw.plan_code,
    planName: raw.plan_name,
    billingCycle: raw.billing_cycle,
    activatedAt: raw.activated_at,
    expiresAt: raw.expires_at,
    usageByDate: {},
  }
}

function normalizeTask(raw: RawTaskView): VIPTaskView {
  return {
    id: raw.id,
    type: raw.type,
    targetType: raw.target_type,
    targetId: raw.target_id,
    targetName: raw.target_name,
    createdAt: raw.created_at,
    status: raw.status,
    startedAt: raw.started_at,
    completedAt: raw.completed_at,
    progressText: raw.progress_text,
    reportId: raw.report_id,
    templateReportId: raw.report_id || '',
  }
}

function normalizeReport(raw?: RawReport | null): VIPReport | null {
  if (!raw) {
    return null
  }

  return {
    ...raw,
    targetName: raw.target_name,
    generatedAt: raw.generated_at,
    coverageWindow: raw.coverage_window,
    footerDisclaimer: raw.footerDisclaimer,
    sources: raw.sources.map((source) => ({
      ...source,
      publishedAt: source.published_at,
    })),
  }
}

function normalizeOrder(raw?: RawOrder | null): VIPOrder | null {
  if (!raw) {
    return null
  }

  return {
    id: raw.id,
    orderNo: raw.order_no,
    planCode: raw.plan_code,
    planName: raw.plan_name,
    billingCycle: raw.billing_cycle,
    amountFen: raw.amount_fen,
    currency: raw.currency,
    status: raw.status,
    paymentChannel: raw.payment_channel,
    paymentScene: raw.payment_scene,
    description: raw.description,
    codeURL: raw.code_url,
    wechatTransactionID: raw.wechat_transaction_id,
    errorCode: raw.error_code,
    errorMessage: raw.error_message,
    expiresAt: raw.expires_at,
    paidAt: raw.paid_at,
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
  }
}

async function fetchVIP<T>(url: string): Promise<T> {
  const res = await fetch(url, {
    credentials: 'include',
  })

  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success || typeof json.data === 'undefined') {
    throw new VIPRequestError(json.error?.message || 'VIP request failed', json.error?.code)
  }

  return json.data
}

async function requestVIP<T>(path: string, init?: RequestInit): Promise<T | null> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success) {
    throw new VIPRequestError(json.error?.message || 'VIP request failed', json.error?.code)
  }

  return json.data ?? null
}

export function useVIPPreview() {
  const { user } = useCurrentUser()

  const membershipKey = user ? `${API_BASE_URL}/api/v1/vip/membership` : null
  const quotaKey = user ? `${API_BASE_URL}/api/v1/vip/quota` : null
  const tasksKey = user ? `${API_BASE_URL}/api/v1/vip/tasks` : null

  const { data: membershipRaw, mutate: mutateMembership } = useSWR<RawMembershipState>(
    membershipKey,
    fetchVIP,
    {
      revalidateOnFocus: false,
      shouldRetryOnError: false,
    }
  )

  const { data: quotaRaw, mutate: mutateQuota } = useSWR<RawQuotaStatus>(
    quotaKey,
    fetchVIP,
    {
      revalidateOnFocus: false,
      shouldRetryOnError: false,
    }
  )

  const { data: tasksRaw = [], mutate: mutateTasks } = useSWR<RawTaskView[]>(
    tasksKey,
    fetchVIP,
    {
      revalidateOnFocus: false,
      refreshInterval: user ? 2000 : 0,
      dedupingInterval: 1000,
      shouldRetryOnError: false,
    }
  )

  const membership = normalizeMembership(membershipRaw)
  const tasks = tasksRaw.map(normalizeTask)
  const latestCompletedTask = tasks.find((task) => task.status === 'completed') ?? null

  const remainingQuota = {
    sectorAnalysis: quotaRaw?.sector_analysis_remaining ?? (membership.isVip ? VIP_DAILY_QUOTA.sectorAnalysis : 0),
    portfolioAnalysis: quotaRaw?.portfolio_analysis_remaining ?? (membership.isVip ? VIP_DAILY_QUOTA.portfolioAnalysis : 0),
  }

  const activateVIP = async (cycle: VIPBillingCycle) => {
    if (!user) {
      return
    }

    await requestVIP('/api/v1/vip/membership/preview-activate', {
      method: 'POST',
      body: JSON.stringify({ billing_cycle: cycle }),
    })

    await Promise.all([mutateMembership(), mutateQuota()])
  }

  const createTask = async (input: {
    type: VIPTaskType
    targetType: VIPTargetType
    targetId: string
    targetName: string
  }) => {
    if (!user) {
      return { ok: false as const, reason: 'not_vip' as const }
    }

    try {
      const result = await requestVIP<{ task_id: string }>('/api/v1/vip/tasks', {
        method: 'POST',
        body: JSON.stringify({
          type: input.type,
          target_type: input.targetType,
          target_id: input.targetId,
          target_name: input.targetName,
        }),
      })

      await Promise.all([mutateTasks(), mutateQuota()])

      return { ok: true as const, taskId: result?.task_id || '' }
    } catch (error) {
      if (error instanceof VIPRequestError) {
        if (error.code === 'VIP_REQUIRED') {
          return { ok: false as const, reason: 'not_vip' as const }
        }
        if (error.code === 'VIP_QUOTA_EXHAUSTED') {
          return { ok: false as const, reason: 'quota_exhausted' as const }
        }
      }
      throw error
    }
  }

  const resetPreview = async () => {
    if (!user) {
      return
    }

    await requestVIP('/api/v1/vip/preview/reset', {
      method: 'POST',
    })

    await Promise.all([mutateMembership(), mutateQuota(), mutateTasks()])
  }

  return {
    membership,
    plan: VIP_PLAN,
    tasks,
    latestCompletedTask,
    remainingQuota,
    canCreateSectorTask: membership.isVip && remainingQuota.sectorAnalysis > 0,
    canCreatePortfolioTask: membership.isVip && remainingQuota.portfolioAnalysis > 0,
    activateVIP,
    createTask,
    resetPreview,
    refreshMembership: mutateMembership,
    refreshQuota: mutateQuota,
    sampleReports: VIP_SAMPLE_REPORTS,
    getReportByID: getVIPSampleReportByID,
  }
}

export function useVIPReport(reportID: string | null) {
  const { data, error, isLoading } = useSWR<RawReport>(
    reportID ? `${API_BASE_URL}/api/v1/vip/reports/${reportID}` : null,
    fetchVIP,
    {
      revalidateOnFocus: false,
      shouldRetryOnError: false,
    }
  )

  return {
    report: normalizeReport(data),
    isLoading,
    error,
  }
}

export async function createVIPOrder(billingCycle: VIPBillingCycle) {
  const data = await requestVIP<RawOrder>('/api/v1/vip/orders', {
    method: 'POST',
    body: JSON.stringify({ billing_cycle: billingCycle }),
  })

  return normalizeOrder(data)
}

export function useVIPOrder(orderID: string | null) {
  const { data, error, isLoading, mutate } = useSWR<RawOrder>(
    orderID ? `${API_BASE_URL}/api/v1/vip/orders/${orderID}` : null,
    fetchVIP,
    {
      revalidateOnFocus: false,
      refreshInterval: orderID ? 3000 : 0,
      dedupingInterval: 1000,
      shouldRetryOnError: false,
    }
  )

  return {
    order: normalizeOrder(data),
    isLoading,
    error,
    refresh: mutate,
  }
}
