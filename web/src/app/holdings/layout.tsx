import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: '个人基金持仓',
  alternates: { canonical: null },
  robots: { index: false, follow: false },
}

export default function HoldingsLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
