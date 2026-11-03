'use client'

import { Children, isValidElement, useEffect, useRef, useState, type ReactNode } from 'react'

export type RevealVariant = 'fade-up' | 'fade-in' | 'scale-in'

export function useLazyReveal<T extends HTMLElement>() {
  const ref = useRef<T | null>(null)
  const [hasEnteredView, setHasEnteredView] = useState(false)

  useEffect(() => {
    if (hasEnteredView) {
      return
    }

    const element = ref.current
    if (!element) {
      return
    }

    if (typeof IntersectionObserver === 'undefined') {
      const frame = window.requestAnimationFrame(() => setHasEnteredView(true))
      return () => window.cancelAnimationFrame(frame)
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting || entry.intersectionRatio > 0) {
          setHasEnteredView(true)
          observer.disconnect()
        }
      },
      {
        rootMargin: '0px 0px 12% 0px',
        threshold: 0,
      }
    )

    observer.observe(element)
    return () => observer.disconnect()
  }, [hasEnteredView])

  return [ref, hasEnteredView] as const
}

// Product data is immediately visible; keep this wrapper API for existing call sites.
export function ScrollReveal({ children, className }: { children: ReactNode; delay?: number; className?: string; variant?: RevealVariant }) {
  return <div className={className}>{children}</div>
}

export function ScrollRevealStack({
  children,
  className,
  itemClassName,
  stagger = 80,
  maxDelay = 360,
}: {
  children: ReactNode
  className?: string
  itemClassName?: string
  stagger?: number
  maxDelay?: number
}) {
  return (
    <div className={className}>
      {Children.map(children, (child, index) => {
        if (!isValidElement(child)) {
          return child
        }

        return (
          <ScrollReveal key={child.key ?? index} delay={Math.min(index * stagger, maxDelay)} className={itemClassName}>
            {child}
          </ScrollReveal>
        )
      })}
    </div>
  )
}
