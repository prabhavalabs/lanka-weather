/**
 * Alert settings, pure: the preferences sent to `/api/push/subscriptions`,
 * their defaults, and what a device can do. Kept free of browser globals so
 * it can be tested.
 */

export const topics = [
  "official",
  "landslide",
  "landslide_level1",
  "river",
  "heavy_rain",
  "flood_risk",
  "flood_forecast",
  "cyclone",
  "earthquake",
  "reservoir",
] as const
export type Topic = (typeof topics)[number]

export const severities = ["watch", "warning", "danger"] as const
export type MinSeverity = (typeof severities)[number]

/** Off by default; the server sends them whatever the minimum severity. */
export const optInTopics: readonly Topic[] = ["landslide_level1", "reservoir"]

export const radii = [5, 10, 25, 50] as const

export interface Near {
  lat: number
  lon: number
  radiusKm: number
}

export interface Preferences {
  topics: Topic[]
  minSeverity: MinSeverity
  areas: { country: boolean; districts: string[]; near: Near | null }
}

/** Official warnings, landslide levels 2 and 3, rivers, heavy rain, the flood-risk estimate, forecasts, cyclones and earthquakes. */
export function defaultPreferences(): Preferences {
  return {
    topics: [
      "official",
      "landslide",
      "river",
      "heavy_rain",
      "flood_risk",
      "flood_forecast",
      "cyclone",
      "earthquake",
    ],
    minSeverity: "warning",
    areas: { country: true, districts: [], near: null },
  }
}

/** Two decimals, about 1.1 km: the most precise location we send. */
export function roundPoint(value: number): number {
  return Math.round(value * 100) / 100
}

const bounds = { south: 5.5, north: 10.5, west: 79, east: 82.5 }

export function inSriLanka(lat: number, lon: number): boolean {
  return lat >= bounds.south && lat <= bounds.north && lon >= bounds.west && lon <= bounds.east
}

/** Why preferences cannot be saved, or null when they can. */
export function problem(preferences: Preferences): "topics" | "areas" | null {
  if (preferences.topics.length === 0) return "topics"
  const { areas } = preferences
  if (!areas.country && areas.districts.length === 0 && !areas.near) return "areas"
  return null
}

/** Reads stored preferences, keeping what is valid and defaulting the rest. */
export function readPreferences(value: unknown): Preferences {
  const fallback = defaultPreferences()
  if (!value || typeof value !== "object") return fallback
  const raw = value as Partial<Preferences>
  const chosen = Array.isArray(raw.topics)
    ? topics.filter((topic) => (raw.topics as unknown[]).includes(topic))
    : fallback.topics
  const minSeverity = severities.includes(raw.minSeverity as MinSeverity)
    ? (raw.minSeverity as MinSeverity)
    : fallback.minSeverity
  const areas = raw.areas && typeof raw.areas === "object" ? raw.areas : fallback.areas
  const districts = Array.isArray(areas.districts)
    ? areas.districts.filter((id): id is string => typeof id === "string" && /^LK-\d{2}$/.test(id))
    : []
  const near =
    areas.near &&
    Number.isFinite(areas.near.lat) &&
    Number.isFinite(areas.near.lon) &&
    inSriLanka(areas.near.lat, areas.near.lon)
      ? {
          lat: roundPoint(areas.near.lat),
          lon: roundPoint(areas.near.lon),
          radiusKm: radii.includes(areas.near.radiusKm as (typeof radii)[number])
            ? areas.near.radiusKm
            : 10,
        }
      : null
  return {
    topics: chosen,
    minSeverity,
    areas: { country: areas.country === true, districts, near },
  }
}

/**
 * The topics there were before the device kept a list of the topics it has
 * seen. Stored settings with no such list were saved with these, so every
 * other default topic is new to them.
 */
export const originalTopics: readonly Topic[] = [
  "official",
  "landslide",
  "landslide_level1",
  "river",
  "flood_forecast",
  "cyclone",
  "earthquake",
  "reservoir",
]

/** The topics a device has been shown, from storage; settings that predate the list saw the original ones. */
export function readKnownTopics(value: unknown): Topic[] {
  if (!Array.isArray(value)) return [...originalTopics]
  return topics.filter((topic) => value.includes(topic))
}

/**
 * Adds every default topic the person has never been shown, so a topic added
 * to the service reaches devices that saved their choices before it existed.
 * A topic they were shown and turned off stays off.
 */
export function withNewTopics(preferences: Preferences, known: readonly Topic[]): Preferences {
  const added = defaultPreferences().topics.filter((topic) => !known.includes(topic))
  if (added.length === 0) return preferences
  return {
    ...preferences,
    topics: topics.filter((topic) => preferences.topics.includes(topic) || added.includes(topic)),
  }
}

/** What the device keeps in localStorage. */
export interface StoredAlerts {
  preferences: Preferences
  enabled: boolean
  /** Every topic the person has been shown, so a new default topic can be told apart from one turned off. */
  known: Topic[]
}

/**
 * Reads the stored alert settings, adding default topics that are new to
 * the person. The result knows every current topic: write it back.
 */
export function readStoredAlerts(value: unknown): StoredAlerts {
  if (!value || typeof value !== "object") {
    return { preferences: defaultPreferences(), enabled: false, known: [...topics] }
  }
  const raw = value as { preferences?: unknown; enabled?: unknown; known?: unknown }
  const known = readKnownTopics(raw.known)
  return {
    preferences: withNewTopics(readPreferences(raw.preferences), known),
    enabled: raw.enabled === true,
    known: [...topics],
  }
}

/** What a device and browser allow. */
export interface Capabilities {
  /** Service workers, the Push API and notifications all exist. */
  push: boolean
  /** An iPhone or iPad, including iPadOS that calls itself a Mac. */
  ios: boolean
  /** Opened from the Home Screen (display-mode standalone). */
  installed: boolean
  permission: NotificationPermission | "unsupported"
}

export type AlertsStatus =
  "checking" | "unsupported" | "ios-install" | "disabled" | "denied" | "unavailable" | "off" | "on"

/** The state the settings show, from what the device allows and the server says. */
export function statusOf(input: {
  capabilities: Capabilities
  /** The server's answer: null while it is being asked. */
  serverEnabled: boolean | null
  subscribed: boolean | null
  workerMissing?: boolean
}): AlertsStatus {
  const { capabilities } = input
  // Safari on iPhone has no Push API in a tab: installing comes first.
  if (capabilities.ios && !capabilities.installed) return "ios-install"
  if (!capabilities.push) return "unsupported"
  if (input.serverEnabled === null) return "checking"
  if (!input.serverEnabled) return "disabled"
  if (capabilities.permission === "denied") return "denied"
  if (input.workerMissing) return "unavailable"
  if (input.subscribed === null) return "checking"
  return input.subscribed ? "on" : "off"
}

export function isIOS(userAgent: string, platform: string, maxTouchPoints: number): boolean {
  return /iPad|iPhone|iPod/.test(userAgent) || (platform === "MacIntel" && maxTouchPoints > 1)
}

export type BrowserFamily = "chrome" | "firefox" | "safari" | "android" | "other"

/** Which re-enabling instructions to show first. */
export function browserFamily(userAgent: string): BrowserFamily {
  if (/Android/.test(userAgent) && !/Firefox/.test(userAgent)) return "android"
  if (/Firefox|FxiOS/.test(userAgent)) return "firefox"
  if (/Edg\/|Chrome\/|CriOS|Chromium/.test(userAgent)) return "chrome"
  if (/Safari\//.test(userAgent)) return "safari"
  return "other"
}

/** A VAPID public key (base64url) as the bytes PushManager.subscribe takes. */
export function keyBytes(base64url: string): Uint8Array<ArrayBuffer> {
  const padded = base64url
    .replace(/-/g, "+")
    .replace(/_/g, "/")
    .padEnd(Math.ceil(base64url.length / 4) * 4, "=")
  const binary = atob(padded)
  const bytes = new Uint8Array(new ArrayBuffer(binary.length))
  for (let index = 0; index < binary.length; index++) bytes[index] = binary.charCodeAt(index)
  return bytes
}

/** Whether a subscription was made with this server key. */
export function sameKey(existing: ArrayBuffer | null, key: Uint8Array): boolean {
  if (!existing) return true
  const bytes = new Uint8Array(existing)
  return bytes.length === key.length && bytes.every((value, index) => value === key[index])
}

/**
 * A Sinhala or Tamil district name without its trailing word "district"
 * (දිස්ත්‍රික්කය, மாவட்டம்), which the regions source often cuts short.
 */
export function shortDistrictName(name: string): string {
  return name.replace(/\s+(දිස්|மாவ)\S*$/u, "").trim() || name
}
