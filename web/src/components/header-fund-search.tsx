'use client'

import { useRouter } from 'next/navigation'
import { FundSearch } from '@/components/fund-search'

interface HeaderFundSearchProps {
  className?: string
}

export function HeaderFundSearch({ className }: HeaderFundSearchProps) {
  const router = useRouter()

  return (
    <FundSearch
      className={className}
      onSelect={(fundId) => {
        router.push(`/?fund=${encodeURIComponent(fundId)}`)
      }}
    />
  )
}
