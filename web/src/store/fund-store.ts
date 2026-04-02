import { create } from 'zustand'

// Types matching backend API
export interface Fund {
  id: string
  name: string
  type: string
  manager: string
  company: string
  nav: string
  scale: string
  updated_at: string
}

export interface HoldingDetail {
  stock_code: string
  stock_name: string
  holding_ratio: string
  stock_change: string
  contribution: string
  current_price: string
  prev_close: string
}

export interface FundEstimate {
  fund_id: string
  fund_name: string
  estimate_nav: string
  prev_nav: string
  change_percent: string
  change_amount: string
  total_hold_ratio: string
  holding_details: HoldingDetail[]
  calculated_at: string
  data_source: string
}

export interface TimeSeriesPoint {
  timestamp: string
  change_percent: string
  estimate_nav: string
}

export interface APIResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
  meta?: {
    data_source?: string
    cache_status?: string
  }
}

// View mode type
export type ViewMode = 'minimal' | 'professional'

// Theme type
export type ThemeType = 'classic' | 'dark' | 'cyber'

// Store state interface
interface FundStore {
  // Current fund data
  currentFundId: string | null
  fund: Fund | null
  estimate: FundEstimate | null
  timeSeries: TimeSeriesPoint[]
  
  // Search
  searchQuery: string
  searchResults: Fund[]
  isSearching: boolean
  
  // UI state
  isLoading: boolean
  error: string | null
  viewMode: ViewMode
  themeType: ThemeType
  lastUpdated: Date | null
  refreshInterval: number // in seconds
  
  // Actions
  setCurrentFundId: (id: string | null) => void
  setFund: (fund: Fund | null) => void
  setEstimate: (estimate: FundEstimate | null) => void
  setTimeSeries: (timeSeries: TimeSeriesPoint[]) => void
  setSearchQuery: (query: string) => void
  setSearchResults: (results: Fund[]) => void
  setIsSearching: (isSearching: boolean) => void
  setIsLoading: (isLoading: boolean) => void
  setError: (error: string | null) => void
  setViewMode: (mode: ViewMode) => void
  setThemeType: (theme: ThemeType) => void
  setLastUpdated: (date: Date | null) => void
  
  // Complex actions
  fetchFundEstimate: (fundId: string) => Promise<void>
  searchFunds: (query: string) => Promise<void>
  fetchTimeSeries: (fundId: string) => Promise<void>
}

// API base URL - in development, this would point to the Go backend
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

// Create the store
export const useFundStore = create<FundStore>((set) => ({
  // Initial state
  currentFundId: null,
  fund: null,
  estimate: null,
  timeSeries: [],
  searchQuery: '',
  searchResults: [],
  isSearching: false,
  isLoading: false,
  error: null,
  viewMode: 'professional',
  themeType: 'dark',
  lastUpdated: null,
  refreshInterval: 60,
  
  // Simple setters
  setCurrentFundId: (id) => set({ currentFundId: id }),
  setFund: (fund) => set({ fund }),
  setEstimate: (estimate) => set({ estimate }),
  setTimeSeries: (timeSeries) => set({ timeSeries }),
  setSearchQuery: (query) => set({ searchQuery: query }),
  setSearchResults: (results) => set({ searchResults: results }),
  setIsSearching: (isSearching) => set({ isSearching }),
  setIsLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),
  setViewMode: (mode) => set({ viewMode: mode }),
  setThemeType: (theme) => set({ themeType: theme }),
  setLastUpdated: (date) => set({ lastUpdated: date }),
  
  // Fetch fund estimate from API
  fetchFundEstimate: async (fundId: string) => {
    set({ isLoading: true, error: null })
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/fund/${fundId}/estimate`)
      const data: APIResponse<FundEstimate> = await response.json()
      
      if (data.success && data.data) {
        set({ 
          estimate: data.data, 
          currentFundId: fundId,
          lastUpdated: new Date(),
          error: null 
        })
        
        // Also fetch fund info
        const fundResponse = await fetch(`${API_BASE_URL}/api/v1/fund/${fundId}`)
        const fundData: APIResponse<Fund> = await fundResponse.json()
        if (fundData.success && fundData.data) {
          set({ fund: fundData.data })
        }
      } else {
        set({ error: data.error?.message || 'Failed to fetch estimate' })
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Network error' })
    } finally {
      set({ isLoading: false })
    }
  },
  
  // Search funds
  searchFunds: async (query: string) => {
    if (!query.trim()) {
      set({ searchResults: [] })
      return
    }
    
    set({ isSearching: true })
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/fund/search?q=${encodeURIComponent(query)}`)
      const data: APIResponse<Fund[]> = await response.json()
      
      if (data.success && data.data) {
        set({ searchResults: data.data })
      } else {
        set({ searchResults: [] })
      }
    } catch (err) {
      console.error('Search error:', err)
      set({ searchResults: [] })
    } finally {
      set({ isSearching: false })
    }
  },
  
  // Fetch time series data
  fetchTimeSeries: async (fundId: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/fund/${fundId}/timeseries`)
      const data: APIResponse<TimeSeriesPoint[]> = await response.json()
      
      if (data.success && data.data) {
        set({ timeSeries: data.data })
      }
    } catch (err) {
      console.error('Time series fetch error:', err)
    }
  },
}))
