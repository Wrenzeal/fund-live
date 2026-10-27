import { redirect } from 'next/navigation'

export default async function AnalysisBoardPage({
  params,
}: {
  params: Promise<{ fundId: string }>
}) {
  const { fundId } = await params

  redirect(`/observations/${encodeURIComponent(fundId)}`)
}
