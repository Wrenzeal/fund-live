'use client'

import { useEffect, useState } from 'react'

const TOP_THRESHOLD_PX = 8

export function useMobileTopSection() {
  const [isAtTop, setIsAtTop] = useState(true)

  useEffect(() => {
    const handleScroll = () => {
      setIsAtTop(window.scrollY <= TOP_THRESHOLD_PX)
    }

    handleScroll()
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const scrollToTop = () => {
    if (typeof window === 'undefined') return
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  return {
    isAtTop,
    showBackToTop: !isAtTop,
    scrollToTop,
  }
}
