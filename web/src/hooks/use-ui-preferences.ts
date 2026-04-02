'use client'

import { useEffect, useState } from 'react'

export type ThemeType = 'classic' | 'dark' | 'cyber'
export type ViewMode = 'minimal' | 'professional'

interface UIPreferencesState {
  themeType: ThemeType
  viewMode: ViewMode
}

const STORAGE_KEY = 'fund-ui-preferences-v1'
const UPDATE_EVENT = 'fund-ui-preferences-updated'

const DEFAULT_STATE: UIPreferencesState = {
  themeType: 'dark',
  viewMode: 'professional',
}

function isBrowser() {
  return typeof window !== 'undefined'
}

function loadPreferences(): UIPreferencesState {
  if (!isBrowser()) {
    return DEFAULT_STATE
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return DEFAULT_STATE
    }

    const parsed = JSON.parse(raw) as Partial<UIPreferencesState>
    return {
      themeType: parsed.themeType ?? DEFAULT_STATE.themeType,
      viewMode: parsed.viewMode ?? DEFAULT_STATE.viewMode,
    }
  } catch {
    return DEFAULT_STATE
  }
}

function savePreferences(state: UIPreferencesState) {
  if (!isBrowser()) {
    return
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  window.dispatchEvent(new CustomEvent(UPDATE_EVENT))
}

export function useUIPreferences() {
  const [state, setState] = useState<UIPreferencesState>(DEFAULT_STATE)

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

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', state.themeType)
  }, [state.themeType])

  const update = (patch: Partial<UIPreferencesState>) => {
    const nextState = {
      ...loadPreferences(),
      ...patch,
    }
    setState(nextState)
    savePreferences(nextState)
  }

  return {
    themeType: state.themeType,
    viewMode: state.viewMode,
    setThemeType: (themeType: ThemeType) => update({ themeType }),
    setViewMode: (viewMode: ViewMode) => update({ viewMode }),
  }
}
