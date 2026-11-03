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
        'flex items-start gap-3 rounded-xl border px-4 py-4 text-sm',
        feedback.type === 'success'
          ? 'border-success/30 bg-success/10 text-success'
          : 'border-warning/30 bg-warning/10 text-warning'
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
