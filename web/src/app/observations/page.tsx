import { getPublicObservationDirectory } from '@/lib/public-observation.server'
import ObservationsPageClient from './page-client'

export const revalidate = 600

export default async function ObservationsPage() {
  const funds = await getPublicObservationDirectory()
  return <ObservationsPageClient directory={funds.map(({ fund, observation }) => ({ id: fund.id, name: fund.name, period: observation.latest_holding_period }))} />
}
