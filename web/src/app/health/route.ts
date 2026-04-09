import type { NextRequest } from 'next/server'
import { proxyToBackend } from '@/lib/backend-proxy'

export function GET(request: NextRequest) {
  return proxyToBackend(request, '/health')
}

export function HEAD(request: NextRequest) {
  return proxyToBackend(request, '/health')
}
