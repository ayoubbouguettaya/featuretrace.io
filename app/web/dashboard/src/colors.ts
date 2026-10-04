// Each service keeps one categorical color for the whole session: slots are
// handed out in a fixed order (8 validated hues, see --series-* in index.css)
// and never reassigned, so filtering never repaints a service. Past eight
// services, the rest share the neutral "other" color.

const SLOT_COUNT = 8
const slots = new Map<string, number>()

/** Assigns slots in list order to services that have none yet. */
export function registerServices(services: string[]): void {
  for (const service of services) serviceSlot(service)
}

/** 1–8, or 0 for "other". */
export function serviceSlot(service: string): number {
  let slot = slots.get(service)
  if (slot === undefined) {
    slot = slots.size < SLOT_COUNT ? slots.size + 1 : 0
    slots.set(service, slot)
  }
  return slot
}

export function serviceColor(service: string): string {
  const slot = serviceSlot(service)
  return slot === 0 ? 'var(--series-other)' : `var(--series-${slot})`
}
