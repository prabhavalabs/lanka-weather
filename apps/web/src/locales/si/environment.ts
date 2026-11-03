import type { Translation } from "@/lib/i18n"
import type { environment as en } from "../en/environment"

export const environment: Translation<typeof en> = {
  aqi: {
    good: "හොඳයි",
    moderate: "මධ්‍යස්ථ",
    unhealthySensitive: "සංවේදී කණ්ඩායම්වලට අහිතකර",
    unhealthy: "අහිතකර",
    veryUnhealthy: "ඉතා අහිතකර",
    hazardous: "අන්තරායකර",
  },
  seaState: {
    calm: "සන්සුන්",
    smooth: "ඉතා මඳ රළ",
    slight: "මඳ රළ",
    moderate: "මධ්‍යස්ථ",
    rough: "රළු",
    veryRough: "ඉතා රළු",
    high: "අධික රළු",
    veryHigh: "අතිශය රළු",
  },
  air: {
    usAqi: "US AQI",
    pm25: "PM2.5",
    legendTitle: "US වායු ගුණත්ව දර්ශකය",
    pm25Title: "PM2.5 · ඇස්තමේන්තුව",
    toggle: "වායු ගුණත්වය පෙන්වන්නේ",
    note: "CAMS වායුගෝල ආකෘතියෙන් ලබාගත් ඇස්තමේන්තුවකි, දළ වශයෙන් 40 km කොටු සහිතය. එය කිසිදු ස්ථානයක වාතය මනින්නේ නැත.",
    credit:
      "Air quality by Open-Meteo.com (CC BY 4.0), generated using Copernicus Atmosphere Monitoring Service information.",
    hoverAqi: "වායු ගුණත්වය (ඇස්තමේන්තුව): {band}, AQI {aqi}",
    hoverPm25: "PM2.5 (ඇස්තමේන්තුව): {value} µg/m³",
  },
  sea: {
    model: "ආකෘතිය",
    waveHeight: "රළ උස",
    legendTitle: "රළ උස · ආකෘතිය",
    note: "0.5° ජාලයක (දළ වශයෙන් 55 km) රළ සහ ඉදිමුම් රළ ආකෘති අගයන්; නිරීක්ෂණ නොවේ. කාලගුණ විද්‍යා දෙපාර්තමේන්තුවේ මුහුදු ප්‍රදේශ අනතුරු ඇඟවීම් අනුගමනය කරන්න.",
    credit: "Marine data by Open-Meteo.com (CC BY 4.0).",
    hover: "රළ (ආකෘතිය): {height} m · {state}",
  },
}
