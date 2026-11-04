import type { Translation } from "@/lib/i18n"
import type { map as en } from "../en/map"

export const map: Translation<typeof en> = {
  label: "இலங்கை வரைபடம்",
  controls: {
    zoomIn: "பெரிதாக்கு",
    zoomOut: "சிறிதாக்கு",
    resetBearing: "திசையை வடக்கிற்கு மீட்டமை",
    resetView: "முழு இலங்கையையும் காட்டு",
    findMyLocation: "எனது இருப்பிடத்தைக் கண்டறி",
    locationNotAvailable: "இருப்பிடம் கிடைக்கவில்லை",
    toggleAttribution: "மூலக் குறிப்புகளைக் காட்டு அல்லது மறை",
    mapFeedback: "வரைபடம் பற்றிய கருத்து",
    map: "வரைபடம்",
  },
  chip: {
    riverGauge: "ஆற்று நீர்மட்ட அளவீட்டு நிலையம்",
    riverGaugeLevel: "{status} · {level}",
    reservoir: "நீர்த்தேக்கம்",
    reservoirStatus: "நீர்த்தேக்கம்: {status}",
    floodForecast: "வெள்ள முன்னறிவிப்பு: {severity}",
    noData: "தரவு இல்லை",
    lowerConfidence: "குறைந்த நம்பகத்தன்மை",
    floodArea: "வெள்ளப் பகுதி",
    googleForecast: "Google வெள்ள முன்னறிவிப்பு",
    stream: "சிற்றாறு",
    tributary: "கிளையாறு, மதிப்பிடப்படவில்லை",
    risk: "{risk} · மதிப்பீடு",
    pastFlood: "கடந்தகால வெள்ளம்",
    pastEvent: "கடந்தகால நிகழ்வு, முன்னறிவிப்பு அல்ல",
    basin: "ஆற்றுப் படுகை",
    basinState: "படுகை: {state}",
    subbasinState: "உப படுகை: {state}",
    regionValue: "{label}: {value} {unit}",
    wind: "காற்று {wind}",
  },
}
