import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: '旧版基金分析',
  alternates: { canonical: null },
  robots: { index: false, follow: false },
}

export default function AnalysisLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
