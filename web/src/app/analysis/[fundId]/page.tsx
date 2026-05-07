import { AnalysisBoardPageClient } from './page-client'

export default async function AnalysisBoardPage({
  params,
}: {
  params: Promise<{ fundId: string }>
}) {
  const { fundId } = await params

  return <AnalysisBoardPageClient fundId={fundId} />
}
