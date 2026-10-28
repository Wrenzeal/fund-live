import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: '账户认证',
  alternates: { canonical: null },
  robots: { index: false, follow: false },
}

export default function AuthLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
