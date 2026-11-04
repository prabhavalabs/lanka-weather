import { translate, type MessageKey } from "@/lib/i18n"
import type { WeatherCondition } from "./types"

const conditionKeys: Record<WeatherCondition, MessageKey> = {
  clear: "weather.condition.clear",
  "clear-night": "weather.condition.clearNight",
  "partly-cloudy": "weather.condition.partlyCloudy",
  cloudy: "weather.condition.cloudy",
  drizzle: "weather.condition.drizzle",
  rain: "weather.condition.rain",
  "heavy-rain": "weather.condition.heavyRain",
  thunderstorm: "weather.condition.thunderstorm",
  fog: "weather.condition.fog",
  windy: "weather.condition.windy",
}

/** Words for a condition in the current language, for labels and screen readers. */
export function conditionLabel(condition: WeatherCondition): string {
  return translate(conditionKeys[condition])
}

/** Words for WMO weather interpretation codes, as Open-Meteo reports them. */
const wmoKeys: Record<number, MessageKey> = {
  0: "weather.wmo.clear",
  1: "weather.wmo.mainlyClear",
  2: "weather.wmo.partlyCloudy",
  3: "weather.wmo.overcast",
  45: "weather.wmo.fog",
  48: "weather.wmo.freezingFog",
  51: "weather.wmo.lightDrizzle",
  53: "weather.wmo.drizzle",
  55: "weather.wmo.denseDrizzle",
  56: "weather.wmo.freezingDrizzle",
  57: "weather.wmo.freezingDrizzle",
  61: "weather.wmo.lightRain",
  63: "weather.wmo.rain",
  65: "weather.wmo.heavyRain",
  66: "weather.wmo.freezingRain",
  67: "weather.wmo.freezingRain",
  71: "weather.wmo.lightSnow",
  73: "weather.wmo.snow",
  75: "weather.wmo.heavySnow",
  77: "weather.wmo.snowGrains",
  80: "weather.wmo.lightShowers",
  81: "weather.wmo.showers",
  82: "weather.wmo.violentShowers",
  85: "weather.wmo.snowShowers",
  86: "weather.wmo.snowShowers",
  95: "weather.wmo.thunderstorm",
  96: "weather.wmo.thunderstormHail",
  99: "weather.wmo.thunderstormHail",
}

/** Sustained wind from which calm skies are called windy, in km/h. */
const windyKmh = 40

/**
 * The condition for a WMO weather code (Open-Meteo's `weatherCode`), the
 * time of day and, optionally, the wind: clear skies at night are
 * "clear-night", and dry weather in a strong wind is "windy". The summary is
 * in the current language, so callers that keep it must depend on the locale.
 */
export function conditionFromWmo(
  code: number | null | undefined,
  isDay: number | boolean | null | undefined = 1,
  windSpeedKmh?: number | null,
): { condition: WeatherCondition; summary: string } {
  const day = isDay === null || isDay === undefined ? true : Boolean(isDay)
  if (code === null || code === undefined || !Number.isFinite(code)) {
    return {
      condition: day ? "partly-cloudy" : "clear-night",
      summary: translate("weather.wmo.unknown"),
    }
  }
  const summary = translate(wmoKeys[code] ?? "weather.wmo.unknown")
  let condition: WeatherCondition
  if (code >= 95) condition = "thunderstorm"
  else if (code === 65 || code === 82 || code === 67) condition = "heavy-rain"
  else if ((code >= 61 && code <= 66) || (code >= 80 && code <= 81)) condition = "rain"
  else if (code >= 51 && code <= 57) condition = "drizzle"
  else if (code === 45 || code === 48) condition = "fog"
  else if (code >= 71 && code <= 86) condition = "cloudy"
  else if (code === 3) condition = "cloudy"
  else if (code === 2) condition = "partly-cloudy"
  else condition = day ? (code === 0 ? "clear" : "partly-cloudy") : "clear-night"
  if (
    code <= 3 &&
    windSpeedKmh !== undefined &&
    windSpeedKmh !== null &&
    windSpeedKmh >= windyKmh
  ) {
    return { condition: "windy", summary: translate("weather.wmo.windy", { summary }) }
  }
  return { condition, summary }
}
