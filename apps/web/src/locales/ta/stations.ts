import type { Translation } from "@/lib/i18n"
import type { stations as en } from "../en/stations"

export const stations: Translation<typeof en> = {
  reading: {
    threeHourly: "3 மணித்தியால அறிக்கை",
    daily: "தினசரி அறிக்கை",
    none: "அளவீடுகள் இல்லை",
  },
  rainLabel: {
    hours: "மழை, {count} மணித்தியாலம்",
    daily: "மழை, காலை 08:30 வரை 24 மணித்தியாலம்",
  },
  notice: {
    dailyTakeover:
      "3 மணித்தியால அறிக்கைகள் புதுப்பிக்கப்படுவது நின்றுவிட்டதால், நிலையங்கள் தினசரி அறிக்கையைக் காட்டுகின்றன.",
    nothing: "வளிமண்டலவியல் திணைக்களத்தின் அளவீடுகள் எதுவும் இன்னும் சேகரிக்கப்படவில்லை.",
    notCurrent:
      "தற்போதையது அல்ல: வளிமண்டலவியல் திணைக்களத்தின் புதிய அளவீடு {age} பெறப்பட்டது. இந்த அறிக்கைகளுக்கான எமது மூலம் புதுப்பிக்கப்படுவது நின்றுவிட்டது.",
  },
  panel: {
    title: "வானிலை நிலையம்",
    subtitle: "வளிமண்டலவியல் நிலையம்",
    subtitleWmo: "வளிமண்டலவியல் நிலையம் · WMO {wmo}",
    subtitleRain: "வளிமண்டலவியல் மழைமானி நிலையம்",
    close: "வானிலை நிலையத்தை மூடு",
    loading: "நிலையம் ஏற்றப்படுகிறது",
    error: "இந்த நிலையத்தை ஏற்ற முடியவில்லை.",
    observed: "அவதானிக்கப்பட்டது",
    outOfDate: "காலாவதியானது",
    temperature: "வெப்பநிலை",
    dailyMaximum: "அதிகபட்சம், தினசரி அறிக்கை",
    observedRow: "அவதானிக்கப்பட்டது",
    observedStale: "{age} · காலாவதியானது",
    readingRow: "அளவீடு",
    rain24h: "மழை, 24 மணித்தியாலம்",
    humidity: "ஈரப்பதன்",
    maximum: "அதிகபட்சம்",
    minimum: "குறைந்தபட்சம்",
    elevation: "உயரம்",
    position: "அமைவிடம்",
    approximate: "தோராயமானது, அதன் பெயரிலிருந்து",
    history: {
      one: "புதிய அறிக்கை வரையான {count} நாள்",
      other: "புதிய அறிக்கை வரையான {count} நாட்கள்",
    },
    dailyReports: "தினசரி அறிக்கைகள்",
    source: "மூலம்: {meteo}, {copy3h} மற்றும் {copyDaily} பிரதிகள் ஊடாக.",
    meteo: "வளிமண்டலவியல் திணைக்களம், இலங்கை",
  },
  chart: {
    empty: {
      one: "புதிய அறிக்கை வரையான {count} நாளில் 3 மணித்தியால அளவீடுகள் இல்லை.",
      other: "புதிய அறிக்கை வரையான {count} நாட்களில் 3 மணித்தியால அளவீடுகள் இல்லை.",
    },
    label: {
      one: "புதிய அறிக்கை வரையான {count} நாளில் அவதானிக்கப்பட்ட வெப்பநிலை {min} முதல் {max} °C வரை",
      other:
        "புதிய அறிக்கை வரையான {count} நாட்களில் அவதானிக்கப்பட்ட வெப்பநிலை {min} முதல் {max} °C வரை",
    },
    labelRain: {
      one: "புதிய அறிக்கை வரையான {count} நாளில் அவதானிக்கப்பட்ட வெப்பநிலை {min} முதல் {max} °C வரை, 3 மணித்தியாலத்தில் {rain} mm வரை மழையுடன்",
      other:
        "புதிய அறிக்கை வரையான {count} நாட்களில் அவதானிக்கப்பட்ட வெப்பநிலை {min} முதல் {max} °C வரை, 3 மணித்தியாலத்தில் {rain} mm வரை மழையுடன்",
    },
    range: "{min}–{max} °C",
    rangeRain: "{min}–{max} °C · மழை {rain} mm வரை",
  },
  summary: {
    observed: "அவதானிக்கப்பட்டது",
    underKm: "1 km இற்குக் குறைவு",
    km: "{km} km",
    dailyMax: "அதிகபட்சம்",
    rainIn: "{hours} மணித்தியாலத்தில் {rain} mm",
    airQuality: "காற்றின் தரம்",
    aqiValue: "{band} · AQI {aqi}",
    sea: "கடல்",
    seaValue: "{state} · அலைகள் {height} m",
    swell: "நீள் அலை {height} m",
    swellPeriod: "நீள் அலை {height} m, {period} s",
    swellFrom: "{direction} திசையிலிருந்து நீள் அலை {height} m",
    swellPeriodFrom: "{direction} திசையிலிருந்து நீள் அலை {height} m, {period} s",
    seaNote:
      "கடல் பெறுமதிகள் அலை மாதிரியிலிருந்து பெறப்பட்டவை. வளிமண்டலவியல் திணைக்களத்தின் கடல் பிராந்திய எச்சரிக்கைகளைப் பின்பற்றவும்.",
  },
  chip: {
    title: "வானிலை நிலையம்",
    observed: "அவதானிக்கப்பட்டது: {temperature} °C",
    dailyMaximum: "அவதானிக்கப்பட்டது, தினசரி அதிகபட்சம்: {temperature} °C",
    ageStale: "{age} · காலாவதியானது",
  },
  layer: {
    observed: "அவதானிக்கப்பட்டது",
    count: { one: "{count} நிலையம்", other: "{count} நிலையங்கள்" },
    legendText:
      "வளிமண்டலவியல் நிலையங்களில் அளவிடப்பட்டது, வெப்பநிலைக்கேற்ப நிறமிடப்பட்டது. மங்கிய சதுரங்கள் காலாவதியானவை.",
    legendTitle: "வெப்பநிலை",
    legendTitleObserved: "வெப்பநிலை, அவதானிக்கப்பட்டது",
  },
}
