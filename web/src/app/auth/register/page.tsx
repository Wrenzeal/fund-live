import { RegisterPageClient } from './page-client'
import { normalizeAuthReturnTo } from '@/lib/auth-return-to'

interface RegisterPageProps {
  searchParams: Promise<{ returnTo?: string | string[] }>
}

export default async function RegisterPage({ searchParams }: RegisterPageProps) {
  const params = await searchParams
  const rawReturnTo = Array.isArray(params.returnTo) ? params.returnTo[0] : params.returnTo
  return <RegisterPageClient returnTo={normalizeAuthReturnTo(rawReturnTo)} />
}
