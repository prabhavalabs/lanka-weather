import type { Translation } from "@/lib/i18n"
import type { map as en } from "../en/map"

export const map: Translation<typeof en> = {
  label: "ශ්‍රී ලංකා සිතියම",
  controls: {
    zoomIn: "විශාලනය කරන්න",
    zoomOut: "කුඩා කරන්න",
    resetBearing: "දිශානතිය උතුරට යළි සකසන්න",
    resetView: "මුළු ශ්‍රී ලංකාවම පෙන්වන්න",
    findMyLocation: "මගේ ස්ථානය සොයන්න",
    locationNotAvailable: "ස්ථානය ලබාගත නොහැක",
    toggleAttribution: "මූලාශ්‍ර සඳහන පෙන්වන්න හෝ සඟවන්න",
    mapFeedback: "සිතියම පිළිබඳ ප්‍රතිචාර",
    map: "සිතියම",
  },
  chip: {
    riverGauge: "ගංගා ජල මාපකය",
    riverGaugeLevel: "{status} · {level}",
    reservoir: "ජලාශය",
    reservoirStatus: "ජලාශය: {status}",
    floodForecast: "ගංවතුර පුරෝකථනය: {severity}",
    noData: "දත්ත නැත",
    lowerConfidence: "අඩු විශ්වාසනීයත්වය",
    floodArea: "ගංවතුර ප්‍රදේශය",
    googleForecast: "Google ගංවතුර පුරෝකථනය",
    stream: "ඇළ දොළ",
    tributary: "අතු ගංගාව, තක්සේරු කර නැත",
    risk: "{risk} · ඇස්තමේන්තුව",
    pastFlood: "අතීත ගංවතුර",
    pastEvent: "අතීත සිදුවීමකි, පුරෝකථනයක් නොවේ",
    basin: "ගංගා ද්‍රෝණිය",
    basinState: "ද්‍රෝණිය: {state}",
    subbasinState: "උප ද්‍රෝණිය: {state}",
    regionValue: "{label}: {value} {unit}",
    wind: "සුළඟ {wind}",
  },
}
