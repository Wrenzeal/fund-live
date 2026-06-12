import type { NextRequest } from 'next/server'

const backendBaseUrl = (
  process.env.BACKEND_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  'http://127.0.0.1:8080'
).replace(/\/$/, '')

const hopByHopHeaders = new Set([
  'connection',
  'content-length',
  'host',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
])

export function copyProxyHeaders(source: Headers) {
  const headers = new Headers()

  source.forEach((value, key) => {
    if (hopByHopHeaders.has(key.toLowerCase())) {
      return
    }
    headers.append(key, value)
  })

  return headers
}

function proxyErrorResponse(targetPath: string, error: unknown) {
  const message = error instanceof Error ? error.message : 'Unknown proxy error'
  console.error('[backend-proxy] failed to reach backend', {
    backendBaseUrl,
    targetPath,
    message,
  })

  return Response.json(
    {
      success: false,
      error: {
        code: 'BACKEND_UNREACHABLE',
        message: 'Backend service is unreachable from the frontend runtime.',
      },
    },
    { status: 502 }
  )
}

export async function proxyToBackend(request: NextRequest, targetPath: string) {
  const method = request.method.toUpperCase()
  const init: RequestInit = {
    method,
    headers: copyProxyHeaders(request.headers),
    redirect: 'manual',
    cache: 'no-store',
    signal: request.signal,
  }

  if (method !== 'GET' && method !== 'HEAD') {
    const body = await request.arrayBuffer()
    if (body.byteLength > 0) {
      init.body = body
    }
  }

  try {
    const upstreamResponse = await fetch(`${backendBaseUrl}${targetPath}`, init)
    const responseBody = method === 'HEAD' ? null : await upstreamResponse.arrayBuffer()

    return new Response(responseBody, {
      status: upstreamResponse.status,
      statusText: upstreamResponse.statusText,
      headers: copyProxyHeaders(upstreamResponse.headers),
    })
  } catch (error) {
    return proxyErrorResponse(targetPath, error)
  }
}
