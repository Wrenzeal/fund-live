import { FundObservationPageClient } from './page-client'

export default async function FundObservationPage({ params }: { params: Promise<{ fundId: string }> }) {
  const { fundId } = await params
  return <FundObservationPageClient key={fundId} fundId={fundId} />
}
