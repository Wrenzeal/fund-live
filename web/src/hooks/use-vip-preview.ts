'use client'

import { useEffect, useMemo, useState } from 'react'
import {
  VIP_DAILY_QUOTA,
  VIP_PLAN,
  VIP_SAMPLE_REPORT_IDS,
  VIP_SAMPLE_REPORTS,
  type VIPBillingCycle,
  type VIPMembershipState,
  type VIPTaskRecord,
  type VIPTaskType,
  type VIPTargetType,
  type VIPTaskView,
  getVIPSampleReportByID,
} from '@/mocks/vip'

const MEMBERSHIP_STORAGE_KEY = 'fundlive-vip-preview-membership-v1'
const TASKS_STORAGE_KEY = 'fundlive-vip-preview-tasks-v1'
const UPDATE_EVENT = 'fundlive-vip-preview-updated'

function isBrowser() {
  return typeof window !== 'undefined'
}

function shanghaiDateKey(date: Date) {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

function addDays(date: Date, days: number) {
  const next = new Date(date)
  next.setDate(next.getDate() + days)
  return next
}

function buildDefaultMembership(): VIPMembershipState {
  return {
    isVip: false,
    planCode: 'vip',
    planName: VIP_PLAN.name,
    billingCycle: 'monthly',
    activatedAt: '',
    expiresAt: '',
    usageByDate: {},
  }
}

function loadMembershipState() {
  if (!isBrowser()) {
    return buildDefaultMembership()
  }

  try {
    const raw = window.localStorage.getItem(MEMBERSHIP_STORAGE_KEY)
    if (!raw) {
      return buildDefaultMembership()
    }

    const parsed = JSON.parse(raw) as Partial<VIPMembershipState>
    return {
      ...buildDefaultMembership(),
      ...parsed,
      usageByDate: parsed.usageByDate ?? {},
    }
  } catch {
    return buildDefaultMembership()
  }
}

function saveMembershipState(state: VIPMembershipState) {
  if (!isBrowser()) {
    return
  }

  window.localStorage.setItem(MEMBERSHIP_STORAGE_KEY, JSON.stringify(state))
  window.dispatchEvent(new CustomEvent(UPDATE_EVENT))
}

function loadTaskRecords() {
  if (!isBrowser()) {
    return [] as VIPTaskRecord[]
  }

  try {
    const raw = window.localStorage.getItem(TASKS_STORAGE_KEY)
    if (!raw) {
      return [] as VIPTaskRecord[]
    }

    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed as VIPTaskRecord[] : []
  } catch {
    return [] as VIPTaskRecord[]
  }
}

function saveTaskRecords(tasks: VIPTaskRecord[]) {
  if (!isBrowser()) {
    return
  }

  window.localStorage.setItem(TASKS_STORAGE_KEY, JSON.stringify(tasks))
  window.dispatchEvent(new CustomEvent(UPDATE_EVENT))
}

function resolveTaskTemplate(type: VIPTaskType, targetName: string) {
  if (type === 'sector_analysis' && targetName.includes('医')) {
    return VIP_SAMPLE_REPORT_IDS.defaultSectorMedical
  }
  if (type === 'sector_analysis') {
    return VIP_SAMPLE_REPORT_IDS.defaultSector
  }
  return VIP_SAMPLE_REPORT_IDS.defaultPortfolio
}

function deriveTaskView(task: VIPTaskRecord, now: number): VIPTaskView {
  const createdAtMs = new Date(task.createdAt).getTime()
  const ageMs = Math.max(0, now - createdAtMs)

  if (ageMs < 3000) {
    return {
      ...task,
      status: 'queued',
      progressText: '已提交，正在整理分析对象与数据上下文',
    }
  }

  if (ageMs < 9000) {
    return {
      ...task,
      status: 'running',
      startedAt: new Date(createdAtMs + 3000).toISOString(),
      progressText: '正在整合宏观、政策、财报与市场走势信息',
    }
  }

  return {
    ...task,
    status: 'completed',
    startedAt: new Date(createdAtMs + 3000).toISOString(),
    completedAt: new Date(createdAtMs + 9000).toISOString(),
    progressText: '报告已生成，可查看完整内容',
    reportId: task.templateReportId,
  }
}

function countUsage(tasks: VIPTaskRecord[], membership: VIPMembershipState) {
  const key = shanghaiDateKey(new Date())
  const storedUsage = membership.usageByDate[key] ?? {
    sectorAnalysis: 0,
    portfolioAnalysis: 0,
  }

  return {
    sectorAnalysis: storedUsage.sectorAnalysis,
    portfolioAnalysis: storedUsage.portfolioAnalysis,
  }
}

function canCreateTask(type: VIPTaskType, membership: VIPMembershipState, tasks: VIPTaskRecord[]) {
  if (!membership.isVip) {
    return false
  }

  const usage = countUsage(tasks, membership)
  if (type === 'sector_analysis') {
    return usage.sectorAnalysis < VIP_DAILY_QUOTA.sectorAnalysis
  }
  return usage.portfolioAnalysis < VIP_DAILY_QUOTA.portfolioAnalysis
}

function updateUsage(type: VIPTaskType, membership: VIPMembershipState) {
  const key = shanghaiDateKey(new Date())
  const current = membership.usageByDate[key] ?? {
    sectorAnalysis: 0,
    portfolioAnalysis: 0,
  }

  membership.usageByDate = {
    ...membership.usageByDate,
    [key]: {
      sectorAnalysis: current.sectorAnalysis + (type === 'sector_analysis' ? 1 : 0),
      portfolioAnalysis: current.portfolioAnalysis + (type === 'portfolio_analysis' ? 1 : 0),
    },
  }
}

export function useVIPPreview() {
  const [membership, setMembership] = useState<VIPMembershipState>(buildDefaultMembership())
  const [taskRecords, setTaskRecords] = useState<VIPTaskRecord[]>([])
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const sync = () => {
      setMembership(loadMembershipState())
      setTaskRecords(loadTaskRecords())
      setNow(Date.now())
    }

    sync()
    window.addEventListener('storage', sync)
    window.addEventListener(UPDATE_EVENT, sync)

    const timer = window.setInterval(() => {
      setNow(Date.now())
    }, 1000)

    return () => {
      window.removeEventListener('storage', sync)
      window.removeEventListener(UPDATE_EVENT, sync)
      window.clearInterval(timer)
    }
  }, [])

  const tasks = useMemo(
    () => taskRecords
      .map((task) => deriveTaskView(task, now))
      .sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime()),
    [now, taskRecords]
  )

  const usage = useMemo(() => countUsage(taskRecords, membership), [membership, taskRecords])
  const remainingQuota = useMemo(() => ({
    sectorAnalysis: Math.max(0, VIP_DAILY_QUOTA.sectorAnalysis - usage.sectorAnalysis),
    portfolioAnalysis: Math.max(0, VIP_DAILY_QUOTA.portfolioAnalysis - usage.portfolioAnalysis),
  }), [usage])

  const latestCompletedTask = useMemo(
    () => tasks.find((task) => task.status === 'completed') ?? null,
    [tasks]
  )

  const activateVIP = (cycle: VIPBillingCycle) => {
    const nowDate = new Date()
    const expiresAt = cycle === 'yearly' ? addDays(nowDate, 365) : addDays(nowDate, 30)
    const nextState: VIPMembershipState = {
      ...membership,
      isVip: true,
      billingCycle: cycle,
      activatedAt: nowDate.toISOString(),
      expiresAt: expiresAt.toISOString(),
    }
    saveMembershipState(nextState)
  }

  const createTask = (input: {
    type: VIPTaskType
    targetType: VIPTargetType
    targetId: string
    targetName: string
  }) => {
    const currentMembership = loadMembershipState()
    const currentTasks = loadTaskRecords()

    if (!currentMembership.isVip) {
      return { ok: false as const, reason: 'not_vip' as const }
    }
    if (!canCreateTask(input.type, currentMembership, currentTasks)) {
      return { ok: false as const, reason: 'quota_exhausted' as const }
    }

    updateUsage(input.type, currentMembership)
    saveMembershipState(currentMembership)

    const nextTask: VIPTaskRecord = {
      id: `vip-task-${Date.now()}`,
      type: input.type,
      targetType: input.targetType,
      targetId: input.targetId,
      targetName: input.targetName,
      createdAt: new Date().toISOString(),
      templateReportId: resolveTaskTemplate(input.type, input.targetName),
    }

    saveTaskRecords([nextTask, ...currentTasks])
    return { ok: true as const, taskId: nextTask.id }
  }

  const resetPreview = () => {
    saveMembershipState(buildDefaultMembership())
    saveTaskRecords([])
  }

  return {
    membership,
    plan: VIP_PLAN,
    tasks,
    latestCompletedTask,
    remainingQuota,
    canCreateSectorTask: canCreateTask('sector_analysis', membership, taskRecords),
    canCreatePortfolioTask: canCreateTask('portfolio_analysis', membership, taskRecords),
    activateVIP,
    createTask,
    resetPreview,
    sampleReports: VIP_SAMPLE_REPORTS,
    getReportByID: getVIPSampleReportByID,
  }
}
