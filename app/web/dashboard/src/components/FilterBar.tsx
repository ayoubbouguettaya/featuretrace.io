import { useCallback, useEffect, useState } from 'react'
import type { Facets } from '../api.ts'
import { LEVELS, RANGES, isDefault, type Filters } from '../filters.ts'

interface Props {
  filters: Filters
  facets: Facets
  onChange: (patch: Partial<Filters>) => void
  onClear: () => void
}

export function FilterBar({ filters, facets, onChange, onClear }: Props) {
  const commitSearch = useCallback((search: string) => onChange({ search }), [onChange])

  return (
    <div className="filter-bar" role="search">
      <SearchInput value={filters.search} onCommit={commitSearch} />

      <FacetSelect
        label="Service"
        allLabel="All services"
        value={filters.service}
        options={facets.services}
        onChange={(service) => onChange({ service })}
      />
      <FacetSelect
        label="Feature"
        allLabel="All features"
        value={filters.feature}
        options={facets.features}
        onChange={(feature) => onChange({ feature })}
      />

      <div className="segmented" role="group" aria-label="Level">
        {['', ...LEVELS].map((level) => (
          <button
            key={level || 'all'}
            type="button"
            aria-pressed={filters.level === level}
            onClick={() => onChange({ level })}
          >
            {level ? <span className={`level level-${level}`}><i aria-hidden="true" />{level}</span> : 'All levels'}
          </button>
        ))}
      </div>

      <label className="field">
        <span className="visually-hidden">Time range</span>
        <select value={filters.range} onChange={(e) => onChange({ range: e.target.value })}>
          {RANGES.map((r) => (
            <option key={r.id} value={r.id}>
              {r.label}
            </option>
          ))}
        </select>
      </label>

      {!isDefault(filters) && (
        <button type="button" className="ghost-button" onClick={onClear}>
          Clear filters
        </button>
      )}
    </div>
  )
}

const SEARCH_DEBOUNCE_MS = 300

/** Text search that commits after the user stops typing. */
function SearchInput({ value, onCommit }: { value: string; onCommit: (value: string) => void }) {
  const [text, setText] = useState(value)
  // Follow changes made elsewhere (Clear filters, back button).
  const [committed, setCommitted] = useState(value)
  if (value !== committed) {
    setCommitted(value)
    setText(value)
  }

  useEffect(() => {
    if (text === value) return
    const timer = window.setTimeout(() => onCommit(text.trim()), SEARCH_DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
  }, [text, value, onCommit])

  return (
    <label className="field search">
      <span className="visually-hidden">Search messages</span>
      <input
        type="search"
        placeholder="Search messages…"
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
    </label>
  )
}

interface FacetSelectProps {
  label: string
  allLabel: string
  value: string
  options: string[]
  onChange: (value: string) => void
}

function FacetSelect({ label, allLabel, value, options, onChange }: FacetSelectProps) {
  // Keep a value from the URL selectable even if it is not (yet) a facet.
  const all = value && !options.includes(value) ? [value, ...options] : options
  return (
    <label className="field">
      <span className="visually-hidden">{label}</span>
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">{allLabel}</option>
        {all.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </label>
  )
}
