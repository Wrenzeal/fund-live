export type AnnouncementSourceType = 'manual' | 'changelog'

export interface Announcement {
  id: string
  title: string
  summary: string
  content: string
  source_type: AnnouncementSourceType
  source_ref: string
  published_at: string
  created_at: string
  updated_at: string
  read?: boolean
}
