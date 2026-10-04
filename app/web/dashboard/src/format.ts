import type { LogRecord } from './api.ts'

const timeFormat = new Intl.DateTimeFormat(undefined, {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})
const dayFormat = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' })

/** "14:03:07.241", prefixed with the day when it is not today. */
export function formatTime(iso: string, now: Date): string {
  const date = new Date(iso)
  const ms = String(date.getMilliseconds()).padStart(3, '0')
  const time = `${timeFormat.format(date)}.${ms}`
  return date.toDateString() === now.toDateString() ? time : `${dayFormat.format(date)} ${time}`
}

/** "0.4 ms", "38.2 ms", "1.24 s". */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms.toFixed(ms < 10 ? 1 : 0)} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

export function shortId(id: string): string {
  return id.length > 8 ? id.slice(0, 8) : id
}

export function isErrorLevel(level: string): boolean {
  return level === 'error' || level === 'fatal'
}

export interface Row {
  key: string
  record: LogRecord
}

/**
 * Records have no id, so rows are keyed by their content (plus a counter for
 * exact duplicates). Keys stay stable across polls, which keeps an expanded
 * row open and lets new rows be highlighted.
 */
export function toRows(records: LogRecord[]): Row[] {
  const seen = new Map<string, number>()
  return records.map((record) => {
    const base = [record.timestamp, record.service, record.span_id, record.message].join('|')
    const n = seen.get(base) ?? 0
    seen.set(base, n + 1)
    return { key: `${base}#${n}`, record }
  })
}
