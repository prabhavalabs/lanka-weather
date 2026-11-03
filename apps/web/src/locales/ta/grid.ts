import type { Translation } from "@/lib/i18n"
import type { grid as en } from "../en/grid"

export const grid: Translation<typeof en> = {
  variable: {
    temperature: { label: "வெப்பநிலை", short: "வெப்பநிலை" },
    precipitation: { label: "இந்த மணித்தியால மழை", short: "மழை" },
    humidity: { label: "சார்பு ஈரப்பதன்", short: "ஈரப்பதன்" },
    cloud: { label: "முகில் மூட்டம்", short: "முகில்" },
    wind: { label: "காற்றின் வேகம்", short: "காற்று" },
  },
  legend: {
    temperature: "தற்போதைய வெப்பநிலை",
    precipitation: "இந்த மணித்தியால மழை",
    light: "சிறிதளவு",
    heavy: "பலத்த மழை",
    violent: "மிகப் பலத்த மழை",
    humidity: "சார்பு ஈரப்பதன்",
    humidityLow: "மணல் நிறம்",
    humidityHigh: "நீலம்",
    cloud: "முகில் மூட்டம்",
    cloudLow: "தெளிவு",
    cloudHigh: "முழு முகில் மூட்டம்",
    wind: "தற்போதைய காற்றின் வேகம்",
  },
  wind: {
    calm: "காற்று மிகக் குறைவு",
    text: "{direction}த் திசையிலிருந்து {kmh} km/h ({knots} kt)",
  },
  timeline: {
    section: "காற்று முன்னறிவிப்பு நேரம்",
    play: "காற்று முன்னறிவிப்பை இயக்கு",
    pause: "காற்று முன்னறிவிப்பை இடைநிறுத்து",
    thumb: "முன்னறிவிப்பு நேரம்",
    thumbValue: "{time}, {lead}",
    scale: "காற்றின் வேகத்திற்கேற்ப துகள்களின் நிறங்கள், km/h",
    source: "தரையிலிருந்து 10 m காற்று · NOAA GFS",
    sourceLoading: "தரையிலிருந்து 10 m காற்று · NOAA GFS · ஏற்றப்படுகிறது…",
    run: "மாதிரி ஓட்டம் {time}",
    now: "இப்போது",
    leadHours: "+{hours} மணித்தியாலம்",
    leadDays: "+{days} நாள்",
    leadDaysHours: "+{days} நாள் {hours} மணித்தியாலம்",
  },
}
