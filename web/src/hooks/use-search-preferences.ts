'use client'

import { useEffect, useState } from 'react'
import type { Fund } from './use-fund-data'

const STORAGE_KEY = 'fund-search-preferences-v1'
const UPDATE_EVENT = 'fund-search-preferences-updated'
const MAX_RECENT = 3
const MAX_QUICK = 3

export interface SearchFundSnapshot {
  id: string
  name: string
  type?: string
  manager?: string
  company?: string
}

interface SearchRecord extends SearchFundSnapshot {
  count: number
  lastSearchedAt: string
}

interface SearchPreferencesState {
  recentIds: string[]
  records: Record<string, SearchRecord>
}

const DEFAULT_STATE: SearchPreferencesState = {
  recentIds: [],
  records: {},
}

function isBrowser() {
  return typeof window !== 'undefined'
}

function loadPreferences(): SearchPreferencesState {
  if (!isBrowser()) {
    return DEFAULT_STATE
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return DEFAULT_STATE
    }

    const parsed = JSON.parse(raw) as Partial<SearchPreferencesState>
    return {
      recentIds: Array.isArray(parsed.recentIds) ? parsed.recentIds : [],
      records: parsed.records ?? {},
    }
  } catch {
    return DEFAULT_STATE
  }
}

function savePreferences(state: SearchPreferencesState) {
  if (!isBrowser()) {
    return
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  window.dispatchEvent(new CustomEvent(UPDATE_EVENT))
}

function toSearchRecord(fund: SearchFundSnapshot, current?: SearchRecord): SearchRecord {
  return {
    id: fund.id,
    name: fund.name,
    type: fund.type ?? current?.type ?? '',
    manager: fund.manager ?? current?.manager ?? '',
    company: fund.company ?? current?.company ?? '',
    count: (current?.count ?? 0) + 1,
    lastSearchedAt: new Date().toISOString(),
  }
}

export function recordFundSelection(fund: SearchFundSnapshot) {
  const current = loadPreferences()
  const nextRecord = toSearchRecord(fund, current.records[fund.id])
  const nextState: SearchPreferencesState = {
    records: {
      ...current.records,
      [fund.id]: nextRecord,
    },
    recentIds: [fund.id, ...current.recentIds.filter((id) => id !== fund.id)].slice(0, MAX_RECENT),
  }

  savePreferences(nextState)
  return nextState
}

export function clearRecentSearches() {
  const current = loadPreferences()
  const nextState: SearchPreferencesState = {
    ...current,
    recentIds: [],
  }

  savePreferences(nextState)
  return nextState
}

function mapIdsToRecords(ids: string[], records: Record<string, SearchRecord>) {
  return ids
    .map((id) => records[id])
    .filter((record): record is SearchRecord => Boolean(record))
}

function sortByCount(records: Record<string, SearchRecord>) {
  return Object.values(records).sort((a, b) => {
    if (b.count !== a.count) {
      return b.count - a.count
    }
    return new Date(b.lastSearchedAt).getTime() - new Date(a.lastSearchedAt).getTime()
  })
}

export function useSearchPreferences() {
  const [state, setState] = useState<SearchPreferencesState>(DEFAULT_STATE)

  useEffect(() => {
    const sync = () => {
      setState(loadPreferences())
    }

    sync()

    window.addEventListener('storage', sync)
    window.addEventListener(UPDATE_EVENT, sync)

    return () => {
      window.removeEventListener('storage', sync)
      window.removeEventListener(UPDATE_EVENT, sync)
    }
  }, [])

  const recentFunds = mapIdsToRecords(state.recentIds, state.records)
  const quickSelectFunds = sortByCount(state.records).slice(0, MAX_QUICK)

  return {
    recentFunds,
    quickSelectFunds,
  }
}

export function toSearchSnapshot(fund: Fund | SearchFundSnapshot): SearchFundSnapshot {
  return {
    id: fund.id,
    name: fund.name,
    type: fund.type,
    manager: fund.manager,
    company: fund.company,
  }
}
