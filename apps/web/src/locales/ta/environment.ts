import type { Translation } from "@/lib/i18n"
import type { environment as en } from "../en/environment"

export const environment: Translation<typeof en> = {
  aqi: {
    good: "நன்று",
    moderate: "மிதமானது",
    unhealthySensitive: "உணர்திறன் மிக்க குழுக்களுக்கு ஆரோக்கியமற்றது",
    unhealthy: "ஆரோக்கியமற்றது",
    veryUnhealthy: "மிகவும் ஆரோக்கியமற்றது",
    hazardous: "அபாயகரமானது",
  },
  seaState: {
    calm: "அமைதியானது",
    smooth: "மென்மையானது",
    slight: "சிறிதளவு கொந்தளிப்பு",
    moderate: "மிதமான கொந்தளிப்பு",
    rough: "கொந்தளிப்பு",
    veryRough: "மிகவும் கொந்தளிப்பு",
    high: "உயர் கொந்தளிப்பு",
    veryHigh: "மிக உயர் கொந்தளிப்பு",
  },
  air: {
    usAqi: "US AQI",
    pm25: "PM2.5",
    legendTitle: "US காற்றுத் தரச் சுட்டெண்",
    pm25Title: "PM2.5 · மதிப்பீடு",
    toggle: "காற்றின் தரம் காட்டுவது",
    note: "CAMS வளிமண்டல மாதிரியிலிருந்தான மதிப்பீடு, சுமார் 40 km அளவிலான கட்டங்கள். இது எந்த இடத்திலும் காற்றை அளவிடுவதில்லை.",
    credit:
      "Air quality by Open-Meteo.com (CC BY 4.0), generated using Copernicus Atmosphere Monitoring Service information.",
    hoverAqi: "காற்றின் தரம் (மதிப்பீடு): {band}, AQI {aqi}",
    hoverPm25: "PM2.5 (மதிப்பீடு): {value} µg/m³",
  },
  sea: {
    model: "மாதிரி",
    waveHeight: "அலை உயரம்",
    legendTitle: "அலை உயரம் · மாதிரி",
    note: "0.5° கட்டத்தில் (சுமார் 55 km) அலை மற்றும் நீள் அலை மாதிரிப் பெறுமதிகள்; அவதானிப்புகள் அல்ல. வளிமண்டலவியல் திணைக்களத்தின் கடல் பிராந்திய எச்சரிக்கைகளைப் பின்பற்றவும்.",
    credit: "Marine data by Open-Meteo.com (CC BY 4.0).",
    hover: "அலைகள் (மாதிரி): {height} m · {state}",
  },
}
