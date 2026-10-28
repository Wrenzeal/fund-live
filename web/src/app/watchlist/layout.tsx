import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: '个人自选基金',
  alternates: { canonical: null },
  robots: { index: false, follow: false },
}

export default function WatchlistLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
