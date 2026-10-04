import type { LogQuery } from './api.ts'

export const LEVELS = ['debug', 'info', 'warn', 'error', 'fatal'] as const

export const RANGES = [
  { id: '15m', label: 'Last 15 minutes', ms: 15 * 60_000 },
  { id: '1h', label: 'Last hour', ms: 60 * 60_000 },
  { id: '24h', label: 'Last 24 hours', ms: 24 * 60 * 60_000 },
  { id: '7d', label: 'Last 7 days', ms: 7 * 24 * 60 * 60_000 },
  { id: 'all', label: 'All time', ms: null },
] as const

export interface Filters {
  search: string
  service: string
  feature: string
  level: string
  traceId: string
  range: string
}

export const DEFAULT_FILTERS: Filters = {
  search: '',
  service: '',
  feature: '',
  level: '',
  traceId: '',
  range: '24h',
}

// URL parameter names match the query API's where one exists.
const PARAMS: Record<keyof Filters, string> = {
  search: 'search',
  service: 'service',
  feature: 'feature',
  level: 'level',
  traceId: 'trace_id',
  range: 'range',
}

export function readFilters(search: string): Filters {
  const params = new URLSearchParams(search)
  const filters = { ...DEFAULT_FILTERS }
  for (const key of Object.keys(PARAMS) as (keyof Filters)[]) {
    filters[key] = params.get(PARAMS[key]) ?? DEFAULT_FILTERS[key]
  }
  return filters
}

/** Query string for the filters, leaving out defaults ('' when all are default). */
export function filtersToSearch(filters: Filters): string {
  const params = new URLSearchParams()
  for (const key of Object.keys(PARAMS) as (keyof Filters)[]) {
    if (filters[key] !== DEFAULT_FILTERS[key]) params.set(PARAMS[key], filters[key])
  }
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

export function isDefault(filters: Filters): boolean {
  return filtersToSearch(filters) === ''
}

/**
 * The query API request for the filters. A trace is shown whole, so a
 * trace_id filter ignores every other filter and the time range.
 */
export function toQuery(filters: Filters, limit: number, now: number): LogQuery {
  if (filters.traceId) return { trace_id: filters.traceId, limit }

  const range = RANGES.find((r) => r.id === filters.range)
  return {
    search: filters.search,
    service: filters.service,
    feature: filters.feature,
    level: filters.level,
    from: range?.ms ? new Date(now - range.ms).toISOString() : undefined,
    limit,
  }
}
