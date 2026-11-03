import type { Translation } from "@/lib/i18n"
import type { location as en } from "../en/location"

export const location: Translation<typeof en> = {
  region: {
    province: "{name} மாகாணம்",
    district: "{name} மாவட்டம்",
  },
  offCoast: "கரைக்கு அப்பால்",
  coordinates: "{lat}° {latHemisphere}, {lon}° {lonHemisphere}",
  hemisphere: {
    north: "வ",
    south: "தெ",
    east: "கி",
    west: "மே",
  },
  marker: "தேர்ந்தெடுத்த இடம்: {name}",
  unavailable: {
    outside: "வானிலைத் தகவல் இலங்கைக்கும் அதன் கரையோரக் கடற்பகுதிக்கும் மட்டுமே கிடைக்கும்.",
    loading:
      "வானிலைத் தரவு இன்னும் ஏற்றப்படுகிறது. வானிலைத் தரவுச் சேகரிப்பு இயங்கியதும் இங்கே தோன்றும்.",
    failed: "வானிலைத் தரவை ஏற்ற முடியவில்லை. சிறிது நேரத்தில் மீண்டும் முயற்சிக்கப்படும்.",
  },
  officialUnread: "உத்தியோகபூர்வ எச்சரிக்கைகளை இப்போது வாசிக்க முடியவில்லை.",
  alert: {
    issuer: "Lanka Weather மதிப்பீடு",
    title: {
      rainYellow: "பலத்த மழை பெய்யக்கூடும்",
      rainOrange: "பலத்த மழை பெய்ய அதிக வாய்ப்புள்ளது",
      rainRed: "மிகப் பலத்த மழை பெய்ய அதிக வாய்ப்புள்ளது",
      windYellow: "பலத்த காற்று வீசக்கூடும்",
      windOrange: "பலத்த காற்று வீச அதிக வாய்ப்புள்ளது",
      windRed: "சேதம் விளைவிக்கும் காற்று வீச அதிக வாய்ப்புள்ளது",
      heatYellow: "வெப்பம்: மிகுந்த அவதானம் தேவை",
      heatOrange: "வெப்பம்: அபாயம்",
      heatRed: "வெப்பம்: மிகுந்த அபாயம்",
      riverYellow: "ஆறு எச்சரிக்கை மட்டத்திற்கு மேல்",
      riverOrange: "ஆறு சிறு வெள்ள மட்டத்தில்",
      riverRed: "ஆறு பெரு வெள்ள மட்டத்தில்",
      floodGauge: "அருகில் ஆற்று வெள்ளம் முன்னறிவிக்கப்பட்டுள்ளது",
      floodArea: "முன்னறிவிக்கப்பட்ட வெள்ளப் பகுதிக்குள்",
    },
    when: {
      next24h: "அடுத்த 24 மணிநேரத்தில்",
      tomorrow: "நாளை",
      dayAfter: "நாளை மறுதினம்",
    },
    rain: "{when} சுமார் {value} mm மழை முன்னறிவிக்கப்பட்டுள்ளது (வரம்பு {threshold} mm).",
    gusts:
      "{when} {value} km/h வரையான வேகக் காற்று முன்னறிவிக்கப்பட்டுள்ளது (வரம்பு {threshold} km/h).",
    winds: "{when} {value} km/h வரையான காற்று முன்னறிவிக்கப்பட்டுள்ளது (வரம்பு {threshold} km/h).",
    heat: "வெப்பச் சுட்டெண் {value} °C வரை உயரக்கூடும் (வரம்பு {threshold} °C). நீர் அருந்துங்கள், நிழலில் ஓய்வெடுங்கள், மற்றவர்களின் நலனையும் கவனியுங்கள்.",
    riverYellow:
      "{distance} km தொலைவிலுள்ள {name} அளவீட்டு நிலையம் எச்சரிக்கை மட்டத்திற்கு மேல் உள்ளது.",
    riverOrange: "{distance} km தொலைவிலுள்ள {name} அளவீட்டு நிலையம் சிறு வெள்ள மட்டத்தில் உள்ளது.",
    riverRed: "{distance} km தொலைவிலுள்ள {name} அளவீட்டு நிலையம் பெரு வெள்ள மட்டத்தில் உள்ளது.",
    floodGaugeYellow:
      "{distance} km தொலைவிலுள்ள ஆற்று அளவீட்டு நிலையத்தில் இயல்பை விட அதிகமான வெள்ளத்தை Google முன்னறிவிக்கிறது.",
    floodGaugeOrange:
      "{distance} km தொலைவிலுள்ள ஆற்று அளவீட்டு நிலையத்தில் கடுமையான வெள்ளத்தை Google முன்னறிவிக்கிறது.",
    floodGaugeRed:
      "{distance} km தொலைவிலுள்ள ஆற்று அளவீட்டு நிலையத்தில் மிகக் கடுமையான வெள்ளத்தை Google முன்னறிவிக்கிறது.",
    floodArea: "இவ்விடம் Google Flood Forecasting வெளியிட்ட வெள்ளப் பகுதிக்குள் உள்ளது.",
    basis: {
      forecast: "வானிலை வலையமைப்பிலிருந்து இடைக்கணிக்கப்பட்ட Open-Meteo முன்னறிவிப்பு",
      heat: "வானிலை வலையமைப்பிலிருந்து இடைக்கணிக்கப்பட்ட Open-Meteo முன்னறிவிப்பு; வெப்பநிலை மற்றும் ஈரப்பதனிலிருந்து வெப்பச் சுட்டெண்",
      rivers:
        "நீர்ப்பாசனத் திணைக்கள ஆற்று அளவீட்டு நிலைய வாசிப்பும் அந்நிலையத்தின் சொந்த மட்டங்களும்",
      riversOther: "{source} அளவீட்டு நிலைய வாசிப்பும் அந்நிலையம் வெளியிட்ட மட்டங்களும்",
      floods: "Google Flood Forecasting",
    },
  },
  lists: {
    rainGauge: "அருகிலுள்ள மழைமானி, கடந்த 24 மணிநேரம்",
    rivers: "அருகிலுள்ள ஆறுகள்",
    riverGauges: "அருகிலுள்ள ஆற்று அளவீட்டு நிலையங்கள்",
    canals: "கால்வாய்கள்",
    floodGauges: "அருகிலுள்ள வெள்ள முன்னறிவிப்பு நிலையங்கள்",
    floodAreas: "இங்குள்ள வெள்ளப் பகுதிகள்",
    reservoirs: "அருகில் வான் பாயும் அல்லது நீர் திறந்துவிடும் நீர்த்தேக்கங்கள்",
    distance: "{distance} km",
    forecastFor: "{window} க்கான முன்னறிவிப்பு",
    colomboCanals: "கொழும்பு கால்வாய்கள்",
    canalsReporting: "{total} இல் {reporting} நிலையங்கள் அறிக்கையிடுகின்றன",
    canalsContext: "தகவலுக்கு மட்டும், வெள்ள எச்சரிக்கை அல்ல",
    raised: "{count} உயர்வு",
    noneRaised: "உயர்வு எதுவுமில்லை",
  },
  sources: {
    item: "{label}: {credit}",
    weather: "வானிலை",
    weatherCredit: "{source} (CC BY 4.0), 0.2° மாதிரி வலையமைப்பு",
    observed: "அவதானிப்பு",
    observedCredit: "வளிமண்டலவியல் திணைக்களம், இலங்கை",
    airQuality: "வளித் தரம்",
    sea: "கடல்",
    riverLevels: "ஆற்று நீர் மட்டங்கள்",
    reservoirs: "நீர்த்தேக்கங்கள்",
    reservoirsCredit:
      "நீர்ப்பாசனத் திணைக்களம் மற்றும் மகாவலி அதிகாரசபை, Sri Lanka Water Levels (waterlevels.lk) வழியாக. இது உத்தியோகபூர்வ எச்சரிக்கை அல்ல.",
    floods: "வெள்ள முன்னறிவிப்புகள்",
    official: "உத்தியோகபூர்வ எச்சரிக்கைகள்",
    officialCredit: "NBRO மற்றும் DMC",
    boundaries: "எல்லைகள்",
    boundariesCredit: "நில அளவைத் திணைக்களம் மற்றும் தொகை மதிப்பு புள்ளிவிபரவியல் திணைக்களம்",
  },
}
