import { Fragment, useState, type KeyboardEvent, type ReactNode } from 'react'
import type { LogRecord } from '../api.ts'
import { serviceColor } from '../colors.ts'
import type { Filters } from '../filters.ts'
import { formatDuration, formatTime, isErrorLevel, type Row } from '../format.ts'
import { LevelBadge, ServiceTag } from './Badges.tsx'

interface Props {
  rows: Row[]
  /** Keys of rows that arrived with the latest poll, highlighted briefly. */
  fresh: Set<string>
  now: Date
  /** Set in trace view: times are shown as offsets from this instant (ms epoch). */
  traceStart?: number
  expandedKey: string | null
  onToggle: (key: string) => void
  onFilter: (patch: Partial<Filters>) => void
  onOpenTrace: (traceId: string) => void
}

export function LogTable({ rows, fresh, now, traceStart, expandedKey, onToggle, onFilter, onOpenTrace }: Props) {
  const inTrace = traceStart !== undefined

  function onRowKey(e: KeyboardEvent<HTMLTableRowElement>, key: string) {
    if (e.target !== e.currentTarget) return
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onToggle(key)
    }
  }

  return (
    <div className="table-scroll">
      <table className="logs">
        <thead>
          <tr>
            <th scope="col">{inTrace ? 'Offset' : 'Time'}</th>
            <th scope="col">Level</th>
            <th scope="col">Service</th>
            <th scope="col">Feature</th>
            <th scope="col">Message</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(({ key, record }) => {
            const expanded = key === expandedKey
            const classes = [
              'log-row',
              isErrorLevel(record.level) && 'is-error',
              fresh.has(key) && 'is-fresh',
              expanded && 'is-expanded',
            ]
            return (
              <Fragment key={key}>
                <tr
                  className={classes.filter(Boolean).join(' ')}
                  style={inTrace && record.service ? { ['--lane' as string]: serviceColor(record.service) } : undefined}
                  tabIndex={0}
                  aria-expanded={expanded}
                  onClick={() => onToggle(key)}
                  onKeyDown={(e) => onRowKey(e, key)}
                >
                  <td className="col-time" title={record.timestamp}>
                    {inTrace
                      ? `+${formatDuration(Date.parse(record.timestamp) - traceStart)}`
                      : formatTime(record.timestamp, now)}
                  </td>
                  <td className="col-level">
                    <LevelBadge level={record.level} />
                  </td>
                  <td className="col-service">
                    <ServiceTag
                      service={record.service}
                      onClick={inTrace || !record.service ? undefined : () => onFilter({ service: record.service })}
                    />
                  </td>
                  <td className="col-feature">
                    {record.feature && !inTrace ? (
                      <button
                        type="button"
                        className="link-button"
                        title={`Show only ${record.feature}`}
                        onClick={(e) => {
                          e.stopPropagation()
                          onFilter({ feature: record.feature })
                        }}
                      >
                        {record.feature}
                      </button>
                    ) : (
                      record.feature || <span className="muted">—</span>
                    )}
                  </td>
                  <td className="col-message">{record.message}</td>
                </tr>
                {expanded && (
                  <tr className="details-row">
                    <td colSpan={5}>
                      <LogDetails record={record} inTrace={inTrace} onOpenTrace={onOpenTrace} />
                    </td>
                  </tr>
                )}
              </Fragment>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function LogDetails({
  record,
  inTrace,
  onOpenTrace,
}: {
  record: LogRecord
  inTrace: boolean
  onOpenTrace: (traceId: string) => void
}) {
  const metadata = Object.entries(record.metadata ?? {})
    .filter(([key]) => key !== 'stack')
    .sort(([a], [b]) => a.localeCompare(b))
  const stack = record.metadata?.stack

  return (
    <div className="details">
      <p className="details-message">{record.message}</p>

      <dl className="details-fields">
        <Field label="Timestamp">
          <code>{record.timestamp}</code>
        </Field>
        <Field label="Trace">
          {record.trace_id ? (
            <>
              <code>{record.trace_id}</code>
              {!inTrace && (
                <button type="button" className="chip-button" onClick={() => onOpenTrace(record.trace_id!)}>
                  View trace →
                </button>
              )}
            </>
          ) : (
            '—'
          )}
        </Field>
        <Field label="Span">{record.span_id ? <code>{record.span_id}</code> : '—'}</Field>
        <Field label="Container">{record.container || '—'}</Field>
        <Field label="Stream">{record.source || '—'}</Field>
      </dl>

      {metadata.length > 0 && (
        <table className="metadata">
          <caption>Metadata</caption>
          <tbody>
            {metadata.map(([key, value]) => (
              <tr key={key}>
                <th scope="row">{key}</th>
                <td>{value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {stack && <pre className="stack">{stack}</pre>}

      <CopyButton text={JSON.stringify(record, null, 2)} />
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  )
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard needs a secure context (https or localhost); nothing to do otherwise.
    }
  }

  return (
    <button type="button" className="ghost-button" onClick={copy}>
      {copied ? 'Copied' : 'Copy as JSON'}
    </button>
  )
}
