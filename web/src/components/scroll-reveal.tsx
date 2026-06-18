'use client'

import { Children, isValidElement, useEffect, useRef, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

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
        rootMargin: '0px 0px -12% 0px',
        threshold: 0.24,
      }
    )

    observer.observe(element)
    return () => observer.disconnect()
  }, [hasEnteredView])

  return [ref, hasEnteredView] as const
}

export function ScrollReveal({
  children,
  delay = 0,
  className,
  variant = 'fade-up',
}: {
  children: ReactNode
  delay?: number
  className?: string
  variant?: RevealVariant
}) {
  const [ref, hasEnteredView] = useLazyReveal<HTMLDivElement>()
  const initialVariantClass = {
    'fade-up': 'translate-y-6 opacity-0',
    'fade-in': 'opacity-0',
    'scale-in': 'scale-[0.98] opacity-0',
  }[variant]
  const enteredVariantClass = {
    'fade-up': 'translate-y-0 opacity-100',
    'fade-in': 'opacity-100',
    'scale-in': 'scale-100 opacity-100',
  }[variant]

  return (
    <div
      ref={ref}
      className={cn(
        'transition-[opacity,transform] duration-500 ease-out motion-reduce:translate-y-0 motion-reduce:scale-100 motion-reduce:opacity-100',
        hasEnteredView ? enteredVariantClass : initialVariantClass,
        className
      )}
      style={{ transitionDelay: hasEnteredView ? `${delay}ms` : '0ms' }}
    >
      {children}
    </div>
  )
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
