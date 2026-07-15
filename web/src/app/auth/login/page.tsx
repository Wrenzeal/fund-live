import { LoginPageClient } from './page-client'
import { normalizeAuthReturnTo } from '@/lib/auth-return-to'

interface LoginPageProps {
  searchParams: Promise<{ returnTo?: string | string[] }>
}

export default async function LoginPage({ searchParams }: LoginPageProps) {
  const params = await searchParams
  const rawReturnTo = Array.isArray(params.returnTo) ? params.returnTo[0] : params.returnTo
  return <LoginPageClient returnTo={normalizeAuthReturnTo(rawReturnTo)} />
}
