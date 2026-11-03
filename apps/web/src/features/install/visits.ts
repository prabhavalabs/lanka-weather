/**
 * When to suggest installing, gently: from the second visit, at most three
 * times, two weeks apart, and never again for three months after "Not now".
 * The rule is pure (`shouldOffer`); the record lives in localStorage.
 */

export interface VisitRecord {
  visits: number
  /** Milliseconds since the epoch. */
  lastVisit: number
  offers: number
  lastOffer: number
  declinedAt: number
  installed: boolean
}

const storageKey = "lanka-weather:install"
const day = 24 * 60 * 60 * 1000

/** A visit counts as new after half an hour away. */
export const visitGap = 30 * 60 * 1000
export const offerRules = {
  minimumVisits: 2,
  maximumOffers: 3,
  offerGap: 14 * day,
  declinePause: 90 * day,
}

export const emptyRecord: VisitRecord = {
  visits: 0,
  lastVisit: 0,
  offers: 0,
  lastOffer: 0,
  declinedAt: 0,
  installed: false,
}

/** The record after a page load at `now`. */
export function recordVisit(record: VisitRecord, now: number): VisitRecord {
  if (record.lastVisit && now - record.lastVisit < visitGap) return { ...record, lastVisit: now }
  return { ...record, visits: record.visits + 1, lastVisit: now }
}

export function shouldOffer(record: VisitRecord, now: number): boolean {
  if (record.installed) return false
  if (record.visits < offerRules.minimumVisits) return false
  if (record.offers >= offerRules.maximumOffers) return false
  if (record.declinedAt && now - record.declinedAt < offerRules.declinePause) return false
  if (record.lastOffer && now - record.lastOffer < offerRules.offerGap) return false
  return true
}

function isRecord(value: unknown): value is VisitRecord {
  if (typeof value !== "object" || value === null) return false
  const record = value as Record<string, unknown>
  return (["visits", "lastVisit", "offers", "lastOffer", "declinedAt"] as const).every(
    (key) => typeof record[key] === "number" && Number.isFinite(record[key]),
  )
}

export function readRecord(): VisitRecord {
  try {
    const stored = JSON.parse(localStorage.getItem(storageKey) ?? "null") as unknown
    return isRecord(stored) ? { ...emptyRecord, ...stored } : emptyRecord
  } catch {
    return emptyRecord
  }
}

export function writeRecord(record: VisitRecord) {
  try {
    localStorage.setItem(storageKey, JSON.stringify(record))
  } catch {
    // Storage may be disabled; the prompt then simply does not come back.
  }
}

export function changeRecord(change: (record: VisitRecord) => VisitRecord): VisitRecord {
  const next = change(readRecord())
  writeRecord(next)
  return next
}

export function markInstalled() {
  changeRecord((record) => ({ ...record, installed: true }))
}
