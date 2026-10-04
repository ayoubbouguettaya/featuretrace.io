// Client for the FeatureTrace query API. Paths are relative: nginx (or the
// Vite dev server) proxies /v1 to the query API.

export interface LogRecord {
  timestamp: string
  message: string
  level: string
  service?: string
  feature?: string
  trace_id?: string
  span_id?: string
  source?: string
  container?: string
  metadata?: Record<string, string>
}

export interface Facets {
  services: string[]
  features: string[]
}

export interface LogQuery {
  service?: string
  level?: string
  feature?: string
  trace_id?: string
  search?: string
  from?: string
  limit: number
}

export async function fetchLogs(query: LogQuery, signal: AbortSignal): Promise<LogRecord[]> {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  const body = await getJSON<{ records: LogRecord[] | null }>(`/v1/logs?${params}`, signal)
  // The API encodes an empty result as null.
  return body.records ?? []
}

export function fetchFacets(signal: AbortSignal): Promise<Facets> {
  return getJSON<Facets>('/v1/facets', signal)
}

async function getJSON<T>(url: string, signal: AbortSignal): Promise<T> {
  const res = await fetch(url, { signal })
  if (!res.ok) throw new Error(`query API answered ${res.status} ${res.statusText}`)
  return (await res.json()) as T
}
