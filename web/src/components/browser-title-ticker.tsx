'use client'

import { useEffect } from 'react'

const SCROLL_THRESHOLD = 14
const SCROLL_GAP = '\u00a0\u00a0\u00a0·\u00a0\u00a0\u00a0'
const SCROLL_STEP_INTERVAL_MS = 320
const SCROLL_START_DELAY_MS = 1800
const TITLE_REST_FRAME_COUNT = 10

interface BrowserTitleTickerProps {
  title: string
}

function buildTitleFrames(title: string) {
  const loopUnits = Array.from(`${title}${SCROLL_GAP}`)
  const restFrames = Array.from({ length: TITLE_REST_FRAME_COUNT }, () => title)
  const scrollFrames = loopUnits.slice(1).map((_, index) => {
    const offset = index + 1
    return [...loopUnits.slice(offset), ...loopUnits.slice(0, offset)].join('')
  })

  return [...restFrames, ...scrollFrames]
}

export function BrowserTitleTicker({ title }: BrowserTitleTickerProps) {
  useEffect(() => {
    document.title = title

    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (prefersReducedMotion || title.length <= SCROLL_THRESHOLD) {
      return
    }

    const frames = buildTitleFrames(title)
    let frame = 0
    let intervalId: number | undefined

    const renderFrame = () => {
      document.title = frames[frame % frames.length] || title
      frame += 1
    }

    const stop = () => {
      if (intervalId !== undefined) {
        window.clearInterval(intervalId)
        intervalId = undefined
      }
      document.title = title
      frame = 0
    }

    const start = () => {
      if (intervalId !== undefined || document.hidden) {
        return
      }
      renderFrame()
      intervalId = window.setInterval(renderFrame, SCROLL_STEP_INTERVAL_MS)
    }

    const startDelayId = window.setTimeout(start, SCROLL_START_DELAY_MS)
    const handleVisibilityChange = () => {
      if (document.hidden) {
        stop()
        return
      }
      start()
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      window.clearTimeout(startDelayId)
      stop()
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [title])

  return null
}
