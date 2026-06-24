'use client'

import { useUIPreferences } from '@/hooks/use-ui-preferences'

export function UIPreferencesSync() {
  useUIPreferences()

  return null
}
