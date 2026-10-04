import { serviceColor } from '../colors.ts'

/** Level as a status dot plus its name; the color never carries it alone. */
export function LevelBadge({ level }: { level: string }) {
  return (
    <span className={`level level-${level}`}>
      <i aria-hidden="true" />
      {level || 'none'}
    </span>
  )
}

/** Service name with its categorical swatch. Clickable when onClick is set. */
export function ServiceTag({ service, onClick }: { service?: string; onClick?: () => void }) {
  if (!service) return <span className="muted">—</span>

  const content = (
    <>
      <i className="swatch" style={{ background: serviceColor(service) }} aria-hidden="true" />
      <span className="service-name">{service}</span>
    </>
  )
  if (!onClick) return <span className="service">{content}</span>
  return (
    <button
      type="button"
      className="service link-button"
      title={`Show only ${service}`}
      onClick={(e) => {
        e.stopPropagation()
        onClick()
      }}
    >
      {content}
    </button>
  )
}
