export function normalizeAuthReturnTo(value: string | null | undefined) {
  const candidate = value?.trim() || '/'
  if (
    !candidate.startsWith('/') ||
    candidate.startsWith('//') ||
    candidate.includes('\\') ||
    /[\u0000-\u001f]/u.test(candidate)
  ) {
    return '/'
  }
  return candidate
}

export function authRouteWithReturnTo(route: '/auth/login' | '/auth/register', returnTo: string) {
  const safeReturnTo = normalizeAuthReturnTo(returnTo)
  return `${route}?returnTo=${encodeURIComponent(safeReturnTo)}`
}
