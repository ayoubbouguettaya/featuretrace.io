// Light / dark / follow-the-OS, remembered per browser. The choice is stamped
// on <html data-theme>; index.css keys its dark tokens off that and the OS.

export type Theme = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'featuretrace.theme'

export function loadTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'light' || stored === 'dark') return stored
  } catch {
    // Storage can be unavailable (private mode); fall back to the OS.
  }
  return 'system'
}

export function applyTheme(theme: Theme): void {
  const root = document.documentElement
  if (theme === 'system') delete root.dataset.theme
  else root.dataset.theme = theme
  try {
    if (theme === 'system') localStorage.removeItem(STORAGE_KEY)
    else localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    // Not remembered, still applied.
  }
}

export function nextTheme(theme: Theme): Theme {
  return theme === 'system' ? 'light' : theme === 'light' ? 'dark' : 'system'
}
