import { useCallback, useEffect, useState } from 'react'
import { fetchFacets, fetchLogs, type Facets } from './api.ts'
import { registerServices } from './colors.ts'
import { DEFAULT_FILTERS, filtersToSearch, isDefault, readFilters, toQuery, type Filters } from './filters.ts'
import { toRows, type Row } from './format.ts'
import { LevelBadge } from './components/Badges.tsx'
import { FilterBar } from './components/FilterBar.tsx'
import { LogTable } from './components/LogTable.tsx'
import { Topbar } from './components/Topbar.tsx'
import { TraceView } from './components/TraceView.tsx'
import './App.css'

const PAGE_SIZE = 100
const TRACE_LIMIT = 500
const POLL_MS = 2000
const FACETS_POLL_MS = 10_000

interface Result {
  /** The request this answers; differs from the current one while loading. */
  key: string
  traceId: string
  rows: Row[]
  /** Rows that were not in the previous answer to the same request. */
  fresh: Set<string>
  error: string | null
  fetchedAt: Date
}

const EMPTY_RESULT: Result = {
  key: '',
  traceId: '',
  rows: [],
  fresh: new Set(),
  error: null,
  fetchedAt: new Date(0),
}

export default function App() {
  const [filters, setFilters] = useState<Filters>(() => readFilters(window.location.search))
  const [limit, setLimit] = useState(PAGE_SIZE)
  const [live, setLive] = useState(true)
  const [refreshes, setRefreshes] = useState(0)
  const [expandedKey, setExpandedKey] = useState<string | null>(null)
  const [facets, setFacets] = useState<Facets>({ services: [], features: [] })
  const [result, setResult] = useState<Result>(EMPTY_RESULT)

  const inTrace = filters.traceId !== ''
  const requestLimit = inTrace ? TRACE_LIMIT : limit
  const requestKey = `${filtersToSearch(filters)}|${requestLimit}`

  // Logs: fetch on every filter change, then poll while live.
  useEffect(() => {
    const ctrl = new AbortController()
    let timer: number | undefined

    const run = async () => {
      try {
        const records = await fetchLogs(toQuery(filters, requestLimit, Date.now()), ctrl.signal)
        const rows = toRows(records)
        setResult((prev) => {
          const known = new Set(prev.key === requestKey ? prev.rows.map((r) => r.key) : [])
          const fresh = new Set(known.size > 0 ? rows.filter((r) => !known.has(r.key)).map((r) => r.key) : [])
          return { key: requestKey, traceId: filters.traceId, rows, fresh, error: null, fetchedAt: new Date() }
        })
      } catch (err) {
        if (ctrl.signal.aborted) return
        setResult((prev) => ({
          ...prev,
          key: requestKey,
          traceId: filters.traceId,
          rows: prev.key === requestKey ? prev.rows : [],
          fresh: new Set(),
          error: err instanceof Error ? err.message : String(err),
        }))
      }
      if (live && !ctrl.signal.aborted) timer = window.setTimeout(run, POLL_MS)
    }

    void run()
    return () => {
      ctrl.abort()
      window.clearTimeout(timer)
    }
  }, [filters, requestLimit, requestKey, live, refreshes])

  // Facets feed the service and feature dropdowns.
  useEffect(() => {
    const ctrl = new AbortController()
    const load = () =>
      fetchFacets(ctrl.signal)
        .then((next) => {
          registerServices(next.services)
          setFacets(next)
        })
        .catch(() => {
          // The logs request reports connectivity problems; dropdowns just stay as they are.
        })

    void load()
    const timer = window.setInterval(load, FACETS_POLL_MS)
    return () => {
      ctrl.abort()
      window.clearInterval(timer)
    }
  }, [])

  // Filters live in the URL so a view can be shared or bookmarked.
  useEffect(() => {
    const search = filtersToSearch(filters)
    if (search !== window.location.search) {
      window.history.replaceState(window.history.state, '', search || window.location.pathname)
    }
  }, [filters])

  useEffect(() => {
    const onPop = () => {
      setFilters(readFilters(window.location.search))
      setLimit(PAGE_SIZE)
      setExpandedKey(null)
    }
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  const updateFilters = useCallback((patch: Partial<Filters>) => {
    setFilters((prev) => ({ ...prev, ...patch }))
    setLimit(PAGE_SIZE)
    setExpandedKey(null)
  }, [])

  function openTrace(traceId: string) {
    const next = { ...DEFAULT_FILTERS, traceId }
    // A history entry, so Back returns to the filtered list.
    window.history.pushState({ fromList: true }, '', filtersToSearch(next))
    setFilters(next)
    setExpandedKey(null)
  }

  function closeTrace() {
    if ((window.history.state as { fromList?: boolean } | null)?.fromList) window.history.back()
    else updateFilters({ traceId: '' })
  }

  const loading = result.key !== requestKey
  // While loading, keep showing the previous rows unless we switched between list and trace.
  const shown = result.traceId === filters.traceId ? result.rows : []
  // The API answers newest first; a trace reads top to bottom, oldest first.
  const rows = inTrace ? [...shown].reverse() : shown
  const traceStart = inTrace && rows.length > 0 ? Date.parse(rows[0].record.timestamp) : undefined
  const hasMore = !inTrace && !loading && shown.length >= limit
  const error = result.key === requestKey ? result.error : null

  let countText = `${rows.length} log${rows.length === 1 ? '' : 's'}, ${inTrace ? 'oldest' : 'newest'} first`
  if (loading && rows.length === 0) countText = 'Loading…'

  let emptyText = 'No logs match these filters. Try a wider time range or clear the filters.'
  if (loading) emptyText = 'Loading…'
  else if (isDefault(filters)) emptyText = 'No logs yet. Logs appear here a few seconds after a labelled container writes them.'

  return (
    <div className="app">
      <Topbar
        live={live}
        onLiveChange={setLive}
        onRefresh={() => setRefreshes((n) => n + 1)}
        onHome={() => updateFilters(DEFAULT_FILTERS)}
      />

      <main className="content">
        {inTrace ? (
          <TraceView
            traceId={filters.traceId}
            records={rows.map((r) => r.record)}
            loading={loading}
            onClose={closeTrace}
          />
        ) : (
          <FilterBar
            filters={filters}
            facets={facets}
            onChange={updateFilters}
            onClear={() => updateFilters(DEFAULT_FILTERS)}
          />
        )}

        {error && (
          <div className="banner-error" role="alert">
            <LevelBadge level="error" />
            <span>
              Can't reach the FeatureTrace query API ({error}).{' '}
              {live ? 'Retrying every 2 seconds.' : 'Press Refresh to retry.'}
            </span>
          </div>
        )}

        {(!inTrace || rows.length > 0) && (
        <section className={`card results${loading ? ' is-loading' : ''}`} aria-busy={loading}>
          <div className="results-head">
            <h2>{inTrace ? 'Logs in this trace' : 'Logs'}</h2>
            <span className="muted small">
              {countText}
            </span>
          </div>

          {rows.length > 0 ? (
            <LogTable
              rows={rows}
              fresh={result.fresh}
              now={result.fetchedAt}
              traceStart={traceStart}
              expandedKey={expandedKey}
              onToggle={(key) => setExpandedKey((current) => (current === key ? null : key))}
              onFilter={updateFilters}
              onOpenTrace={openTrace}
            />
          ) : (
            <p className="empty-note">
              {emptyText}
            </p>
          )}

          {hasMore && (
            <div className="results-foot">
              <button type="button" className="ghost-button" onClick={() => setLimit((n) => n + PAGE_SIZE)}>
                Load {PAGE_SIZE} more
              </button>
            </div>
          )}
        </section>
        )}
      </main>
    </div>
  )
}
