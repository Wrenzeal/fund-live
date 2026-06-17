const PRODUCTION_FRONTEND_HOSTS = new Set([
  'fund.wrenzeal.top',
  'www.fund.wrenzeal.top',
])

export const PRODUCTION_API_BASE_URL = 'https://api.fund.wrenzeal.top'

// Browser code should normally call same-origin /api so auth cookies stay on the
// frontend domain. If a public API URL is accidentally set to the Vercel frontend
// host, treat it as unset instead of routing through the frontend domain again.

function stripTrailingSlash(value: string) {
  return value.trim().replace(/\/+$/, '')
}

export function isProductionFrontendUrl(value: string) {
  try {
    const url = new URL(value)
    return PRODUCTION_FRONTEND_HOSTS.has(url.hostname)
  } catch {
    return false
  }
}

export function resolvePublicApiBaseUrl(value = process.env.NEXT_PUBLIC_API_URL || '') {
  const normalized = stripTrailingSlash(value)
  if (!normalized) {
    return ''
  }

  try {
    if (isProductionFrontendUrl(normalized)) {
      return ''
    }
  } catch {
    return normalized
  }

  return normalized
}

export function resolveBackendBaseUrl(value: string | undefined, fallback: string) {
  const normalized = stripTrailingSlash(value || '')
  if (!normalized) {
    return fallback
  }

  if (isProductionFrontendUrl(normalized)) {
    return PRODUCTION_API_BASE_URL
  }

  return normalized
}

export const API_BASE_URL = resolvePublicApiBaseUrl()
