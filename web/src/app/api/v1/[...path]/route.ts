import type { NextRequest } from 'next/server'
import { proxyToBackend } from '@/lib/backend-proxy'

function buildApiPath(pathSegments: string[], search: string) {
  const normalizedPath = pathSegments.map(encodeURIComponent).join('/')
  return `/api/v1/${normalizedPath}${search}`
}

async function handleProxy(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  const { path } = await context.params
  return proxyToBackend(request, buildApiPath(path, request.nextUrl.search))
}

export function GET(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  return handleProxy(request, context)
}

export function POST(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  return handleProxy(request, context)
}

export function PUT(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  return handleProxy(request, context)
}

export function DELETE(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  return handleProxy(request, context)
}

export function PATCH(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  return handleProxy(request, context)
}

export function OPTIONS(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  return handleProxy(request, context)
}

export function HEAD(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  return handleProxy(request, context)
}
