/**
 * Realistic placeholder props for the weather shells, for the design kit
 * (/?kit in development) and tests. Not real observations.
 */
import type {
  AdminArea,
  CurrentConditions,
  ForecastHour,
  Place,
  SearchResult,
  WeatherAlert,
  WeatherCondition,
  WindReading,
} from "./types"

export const samplePlace: Place = {
  name: "Ratnapura",
  area: "Ratnapura District · Sabaragamuwa Province",
  latitude: 6.6828,
  longitude: 80.3992,
}

export const sampleCurrent: CurrentConditions = {
  condition: "partly-cloudy",
  summary: "Partly cloudy",
  temperatureC: 28.8,
  feelsLikeC: 33.6,
  humidityPercent: 84,
  rainfall24hMm: 86.5,
  pressureHpa: 1008,
  observedAt: "2026-11-03T08:30:00Z",
}

export const sampleWind: WindReading = { speedKmh: 24, gustKmh: 46, fromDegrees: 225 }

/** 48 hours from 14:00 Sri Lanka time: warm afternoons, evening thunderstorms. */
export const sampleForecast: ForecastHour[] = Array.from({ length: 48 }, (_, index) => {
  const time = new Date(Date.UTC(2026, 10, 3, 8, 30) + index * 3_600_000)
  const hour = (14 + index) % 24
  const temperatureC = 24 + 5 * Math.sin(((hour - 8) / 24) * 2 * Math.PI) + (index > 24 ? 0.6 : 0)
  const storm = hour >= 15 && hour <= 20
  const rainfallMm = storm
    ? [4.2, 9.8, 14.5, 11.2, 6.1, 2.4][hour - 15]! * (index > 24 ? 0.7 : 1)
    : hour < 6
      ? 0.4
      : 0
  const condition: WeatherCondition = storm
    ? rainfallMm > 10
      ? "thunderstorm"
      : "rain"
    : hour >= 19 || hour < 6
      ? rainfallMm > 0
        ? "drizzle"
        : "clear-night"
      : "partly-cloudy"
  return {
    time: time.toISOString(),
    condition,
    temperatureC: Math.round(temperatureC * 10) / 10,
    rainfallMm: Math.round(rainfallMm * 10) / 10,
    rainChancePercent: storm ? 80 + (hour % 3) * 5 : rainfallMm > 0 ? 40 : 10,
  }
})

export const sampleAlerts: WeatherAlert[] = [
  {
    id: "dmc-landslide-2026-11-03",
    title: "Landslide early warning: level 2 (amber)",
    severity: "warning",
    issuer: "National Building Research Organisation",
    area: "Ratnapura, Kegalle and Kalutara districts",
    issuedAt: "2026-11-03T06:00:00Z",
    validUntil: "2026-11-04T06:00:00Z",
    summary:
      "Over 100 mm of rain in 24 hours. Watch for cracks in walls and ground, and tilting trees.",
  },
  {
    id: "dom-heavy-rain-2026-11-03",
    title: "Heavy rainfall advisory",
    severity: "advisory",
    issuer: "Department of Meteorology",
    area: "Sabaragamuwa, Western and Southern provinces",
    issuedAt: "2026-11-03T04:30:00Z",
    validUntil: "2026-11-03T22:30:00Z",
  },
]

export const sampleProvinces: AdminArea[] = [
  "Western",
  "Central",
  "Southern",
  "Northern",
  "Eastern",
  "North Western",
  "North Central",
  "Uva",
  "Sabaragamuwa",
].map((name) => ({
  id: `province-${name.toLowerCase().replace(/\s/g, "-")}`,
  name,
  level: "province",
}))

export const sampleDistricts: AdminArea[] = [
  { id: "district-ratnapura", name: "Ratnapura", level: "district", parentName: "Sabaragamuwa" },
  { id: "district-kegalle", name: "Kegalle", level: "district", parentName: "Sabaragamuwa" },
  { id: "district-kalutara", name: "Kalutara", level: "district", parentName: "Western" },
  { id: "district-colombo", name: "Colombo", level: "district", parentName: "Western" },
  { id: "district-galle", name: "Galle", level: "district", parentName: "Southern" },
]

export const sampleSearchResults: SearchResult[] = [
  {
    id: "place-ratnapura",
    kind: "place",
    title: "Ratnapura",
    subtitle: "Ratnapura District",
    meta: "26°",
  },
  {
    id: "place-colombo",
    kind: "place",
    title: "Colombo",
    subtitle: "Colombo District",
    meta: "29°",
  },
  {
    id: "station-ratnapura",
    kind: "station",
    title: "Ratnapura",
    subtitle: "Kalu Ganga · river gauge",
    meta: "7.82 m",
  },
  {
    id: "station-hanwella",
    kind: "station",
    title: "Hanwella",
    subtitle: "Kelani Ganga · river gauge",
    meta: "4.10 m",
  },
  {
    id: "area-kuruwita",
    kind: "area",
    title: "Kuruwita",
    subtitle: "DS division · Ratnapura District",
  },
]
