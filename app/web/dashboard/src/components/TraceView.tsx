import { useState } from 'react'
import type { LogRecord } from '../api.ts'
import { serviceColor } from '../colors.ts'
import { formatDuration, isErrorLevel, shortId } from '../format.ts'
import { LevelBadge, ServiceTag } from './Badges.tsx'

interface Props {
  traceId: string
  /** The trace's logs, oldest first. */
  records: LogRecord[]
  loading: boolean
  onClose: () => void
}

/** One service's part of the trace: its logs sharing a span id. */
interface Span {
  key: string
  service: string
  spanId: string
  /** ms from the trace's first log. */
  start: number
  end: number
  logs: number
  errors: number
}

export function TraceView({ traceId, records, loading, onClose }: Props) {
  const [active, setActive] = useState<string | null>(null)

  const header = (
    <div className="trace-head">
      <div>
        <p className="eyebrow">Trace</p>
        <h1 className="trace-id">
          <code>{traceId}</code>
        </h1>
      </div>
      <button type="button" className="ghost-button" onClick={onClose}>
        ← All logs
      </button>
    </div>
  )

  if (records.length === 0) {
    return (
      <section className="card trace-card" aria-label="Trace">
        {header}
        <p className="empty-note">
          {loading
            ? 'Loading trace…'
            : 'No logs for this trace yet. Logs reach FeatureTrace a few seconds after the request; this view refreshes on its own while Live is on.'}
        </p>
      </section>
    )
  }

  const t0 = Date.parse(records[0].timestamp)
  const spans = toSpans(records, t0)
  const total = Math.max(...spans.map((s) => s.end))
  const scale = total > 0 ? total : 1
  const services = new Set(spans.map((s) => s.service)).size
  const errors = records.filter((r) => isErrorLevel(r.level)).length
  const pct = (ms: number) => (ms / scale) * 100

  return (
    <section className="card trace-card" aria-label="Trace">
      {header}

      <dl className="trace-stats">
        <Stat label="Duration" value={formatDuration(total)} />
        <Stat label="Services" value={services} />
        <Stat label="Spans" value={spans.length} />
        <Stat label="Logs" value={records.length} />
        <div>
          <dt>Errors</dt>
          <dd>
            {errors > 0 && <i className="status-dot" aria-hidden="true" />}
            {errors}
          </dd>
        </div>
      </dl>

      <div className="waterfall" role="table" aria-label="Spans over time">
        <div className="wf-row wf-axis" role="row" aria-hidden="true">
          <div className="wf-label" />
          <div className="wf-lane">
            <span style={{ left: '0%' }}>0 ms</span>
            <span style={{ left: '50%' }}>{formatDuration(total / 2)}</span>
            <span style={{ left: '100%' }}>{formatDuration(total)}</span>
          </div>
        </div>

        {spans.map((span) => {
          const left = pct(span.start)
          const isActive = active === span.key
          return (
            <div
              key={span.key}
              className={`wf-row${isActive ? ' is-active' : ''}`}
              role="row"
              tabIndex={0}
              onMouseEnter={() => setActive(span.key)}
              onMouseLeave={() => setActive(null)}
              onFocus={() => setActive(span.key)}
              onBlur={() => setActive(null)}
            >
              <div className="wf-label" role="rowheader">
                <ServiceTag service={span.service} />
                {span.spanId && <code className="muted small">{shortId(span.spanId)}</code>}
                {span.errors > 0 && <LevelBadge level="error" />}
              </div>
              <div className="wf-lane" role="cell">
                <div
                  className="wf-bar"
                  style={{
                    left: `${left}%`,
                    width: `max(8px, ${pct(span.end - span.start)}%)`,
                    background: serviceColor(span.service),
                  }}
                />
                {isActive && (
                  <div
                    className="wf-tooltip"
                    role="tooltip"
                    style={left > 60 ? { right: `${100 - left}%` } : { left: `${left}%` }}
                  >
                    <strong>{span.service}</strong>
                    {span.spanId && <code>{span.spanId}</code>}
                    <span>
                      starts +{formatDuration(span.start)} · lasts {formatDuration(span.end - span.start)}
                    </span>
                    <span>
                      {span.logs} log{span.logs === 1 ? '' : 's'}
                      {span.errors > 0 && ` · ${span.errors} error${span.errors === 1 ? '' : 's'}`}
                    </span>
                  </div>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </section>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  )
}

/** Groups logs by service and span, in order of first appearance. */
function toSpans(records: LogRecord[], t0: number): Span[] {
  const spans = new Map<string, Span>()
  for (const record of records) {
    const t = Date.parse(record.timestamp) - t0
    const service = record.service || 'unknown'
    const spanId = record.span_id ?? ''
    const key = `${service}|${spanId}`
    const span = spans.get(key) ?? { key, service, spanId, start: t, end: t, logs: 0, errors: 0 }
    span.start = Math.min(span.start, t)
    span.end = Math.max(span.end, t)
    span.logs += 1
    if (isErrorLevel(record.level)) span.errors += 1
    spans.set(key, span)
  }
  return [...spans.values()]
}
