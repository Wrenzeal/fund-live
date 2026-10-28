import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: '问题反馈',
  alternates: { canonical: null },
  robots: { index: false, follow: false },
}

export default function IssuesLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
