import { useState } from 'react'
import { applyTheme, loadTheme, nextTheme, type Theme } from '../theme.ts'

interface Props {
  live: boolean
  onLiveChange: (live: boolean) => void
  onRefresh: () => void
  onHome: () => void
}

const THEME_LABELS: Record<Theme, string> = { system: 'Auto', light: 'Light', dark: 'Dark' }

export function Topbar({ live, onLiveChange, onRefresh, onHome }: Props) {
  const [theme, setTheme] = useState<Theme>(loadTheme)

  function cycleTheme() {
    const next = nextTheme(theme)
    applyTheme(next)
    setTheme(next)
  }

  return (
    <header className="topbar">
      <a
        className="brand"
        href="/"
        onClick={(e) => {
          e.preventDefault()
          onHome()
        }}
      >
        <img src="/logo.png" alt="FeatureTrace" width={160} height={34} />
        <span className="brand-section">Logs</span>
      </a>

      <div className="topbar-actions">
        <button
          type="button"
          className={`live-toggle${live ? ' is-live' : ''}`}
          aria-pressed={live}
          title={live ? 'Pause auto-refresh' : 'Refresh every 2 seconds'}
          onClick={() => onLiveChange(!live)}
        >
          <i aria-hidden="true" />
          {live ? 'Live' : 'Paused'}
        </button>
        <button type="button" className="topbar-button" onClick={onRefresh}>
          Refresh
        </button>
        <button type="button" className="topbar-button" title="Switch theme" onClick={cycleTheme}>
          Theme: {THEME_LABELS[theme]}
        </button>
      </div>
    </header>
  )
}
