'use client'

import { useEffect, useState, type ReactNode, type RefObject } from 'react'
import { createPortal } from 'react-dom'

import { cn } from '@/lib/utils'

type FloatingListboxRect = {
  left: number
  top: number
  width: number
  maxHeight: number
}

interface FloatingListboxProps {
  open: boolean
  triggerRef: RefObject<HTMLElement | null>
  ariaLabel: string
  children: ReactNode
  id?: string
  onClose?: () => void
  withBackdrop?: boolean
  minWidth?: number
  minHeight?: number
  maxHeight?: number
  gap?: number
  viewportPadding?: number
  className?: string
  contentClassName?: string
}

export function FloatingListbox({
  open,
  triggerRef,
  ariaLabel,
  children,
  id,
  onClose,
  withBackdrop = false,
  minWidth = 240,
  minHeight = 160,
  maxHeight = 320,
  gap = 10,
  viewportPadding = 16,
  className,
  contentClassName,
}: FloatingListboxProps) {
  const [rect, setRect] = useState<FloatingListboxRect | null>(null)

  useEffect(() => {
    if (!open) {
      return
    }

    let frameID: number | null = null

    const updateRect = () => {
      const trigger = triggerRef.current
      if (!trigger) {
        setRect(null)
        return
      }

      const triggerRect = trigger.getBoundingClientRect()
      const availableWidth = Math.max(minWidth, window.innerWidth - viewportPadding * 2)
      const width = Math.min(triggerRect.width, availableWidth)
      const left = Math.min(
        Math.max(triggerRect.left, viewportPadding),
        Math.max(viewportPadding, window.innerWidth - width - viewportPadding)
      )
      const top = Math.min(triggerRect.bottom + gap, window.innerHeight - viewportPadding)
      const availableHeight = window.innerHeight - top - viewportPadding
      const nextMaxHeight = Math.max(minHeight, Math.min(maxHeight, availableHeight))

      setRect({ left, top, width, maxHeight: nextMaxHeight })
    }

    frameID = window.requestAnimationFrame(updateRect)
    window.addEventListener('resize', updateRect)
    window.addEventListener('scroll', updateRect, true)

    return () => {
      if (frameID !== null) {
        window.cancelAnimationFrame(frameID)
      }
      window.removeEventListener('resize', updateRect)
      window.removeEventListener('scroll', updateRect, true)
    }
  }, [gap, maxHeight, minHeight, minWidth, open, triggerRef, viewportPadding])

  if (!open || !rect || typeof document === 'undefined') {
    return null
  }

  return createPortal(
    <>
      {withBackdrop && <div className="fixed inset-0 z-[80] cursor-default" onClick={onClose} />}
      <div
        id={id}
        className={cn(
          'watchlist-select-dropdown pointer-events-auto fixed z-[90] overflow-hidden rounded-[24px] border border-cyan-400/22 p-2 shadow-[0_24px_60px_rgba(2,8,23,0.42)]',
          className
        )}
        style={{ left: rect.left, top: rect.top, width: rect.width }}
        role="listbox"
        aria-label={ariaLabel}
      >
        <div className={cn('space-y-1 overflow-y-auto', contentClassName)} style={{ maxHeight: rect.maxHeight }}>
          {children}
        </div>
      </div>
    </>,
    document.body
  )
}
