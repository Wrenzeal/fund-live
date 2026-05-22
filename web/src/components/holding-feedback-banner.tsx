'use client'

import { AlertTriangle, CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface HoldingFeedbackMessage {
  type: 'success' | 'error'
  message: string
}

interface HoldingFeedbackBannerProps {
  feedback: HoldingFeedbackMessage
  showIcon?: boolean
}

export function HoldingFeedbackBanner({ feedback, showIcon = true }: HoldingFeedbackBannerProps) {
  return (
    <div
      className={cn(
        'flex items-start gap-3 rounded-[28px] border px-4 py-4 text-sm',
        feedback.type === 'success'
          ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-50'
          : 'border-amber-500/30 bg-amber-500/10 text-amber-100'
      )}
    >
      {showIcon && (
        feedback.type === 'success' ? (
          <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
        ) : (
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        )
      )}
      <span>{feedback.message}</span>
    </div>
  )
}
