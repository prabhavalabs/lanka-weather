import type { Translation } from "@/lib/i18n"
import type { stations as en } from "../en/stations"

export const stations: Translation<typeof en> = {
  reading: {
    threeHourly: "පැය 3 වාර්තාව",
    daily: "දෛනික වාර්තාව",
    none: "කියවීම් නැත",
  },
  rainLabel: {
    hours: "වර්ෂාපතනය, පැය {count}",
    daily: "වර්ෂාපතනය, පෙ.ව. 08:30 දක්වා පැය 24",
  },
  notice: {
    dailyTakeover:
      "පැය 3 වාර්තා යාවත්කාලීන වීම නැවතී ඇති බැවින්, මධ්‍යස්ථාන දෛනික වාර්තාව පෙන්වයි.",
    nothing: "කාලගුණ විද්‍යා දෙපාර්තමේන්තුවේ කියවීම් තවම එක්රැස් කර නැත.",
    notCurrent:
      "යාවත්කාලීන නැත: කාලගුණ විද්‍යා දෙපාර්තමේන්තුවේ නවතම කියවීම ලැබුණේ {age}. මෙම වාර්තා සඳහා අපගේ මූලාශ්‍රය යාවත්කාලීන වීම නැවතී ඇත.",
  },
  panel: {
    title: "කාලගුණ මධ්‍යස්ථානය",
    subtitle: "කාලගුණ මධ්‍යස්ථානය",
    subtitleWmo: "කාලගුණ මධ්‍යස්ථානය · WMO {wmo}",
    subtitleRain: "කාලගුණ දෙපාර්තමේන්තු වර්ෂාමාන මධ්‍යස්ථානය",
    close: "කාලගුණ මධ්‍යස්ථානය වසන්න",
    loading: "මධ්‍යස්ථානය පූරණය වෙමින්",
    error: "මෙම මධ්‍යස්ථානය පූරණය කළ නොහැකි විය.",
    observed: "නිරීක්ෂිත",
    outOfDate: "යල් පැන ගිය",
    temperature: "උෂ්ණත්වය",
    dailyMaximum: "උපරිමය, දෛනික වාර්තාව",
    observedRow: "නිරීක්ෂණය කළේ",
    observedStale: "{age} · යල් පැන ගිය",
    readingRow: "කියවීම",
    rain24h: "වර්ෂාපතනය, පැය 24",
    humidity: "ආර්ද්‍රතාව",
    maximum: "උපරිමය",
    minimum: "අවමය",
    elevation: "උන්නතාංශය",
    position: "පිහිටීම",
    approximate: "ආසන්න වශයෙන්, එහි නම අනුව",
    history: "නවතම වාර්තාව දක්වා දින {count}",
    dailyReports: "දෛනික වාර්තා",
    source: "මූලාශ්‍රය: {meteo}, {copy3h} සහ {copyDaily} පිටපත් හරහා.",
    meteo: "කාලගුණ විද්‍යා දෙපාර්තමේන්තුව, ශ්‍රී ලංකාව",
  },
  chart: {
    empty: "නවතම වාර්තාව දක්වා දින {count} තුළ පැය 3 කියවීම් නැත.",
    label: "නවතම වාර්තාව දක්වා දින {count} තුළ නිරීක්ෂිත උෂ්ණත්වය {min} සිට {max} °C දක්වා",
    labelRain:
      "නවතම වාර්තාව දක්වා දින {count} තුළ නිරීක්ෂිත උෂ්ණත්වය {min} සිට {max} °C දක්වා, පැය 3කදී {rain} mm දක්වා වර්ෂාපතනය සමඟ",
    range: "{min}–{max} °C",
    rangeRain: "{min}–{max} °C · වර්ෂාව {rain} mm දක්වා",
  },
  summary: {
    observed: "නිරීක්ෂිත",
    underKm: "1 km ට අඩු",
    km: "{km} km",
    dailyMax: "උපරිම",
    rainIn: "පැය {hours}කදී {rain} mm",
    airQuality: "වායු ගුණත්වය",
    aqiValue: "{band} · AQI {aqi}",
    sea: "මුහුද",
    seaValue: "{state} · රළ {height} m",
    swell: "ඉදිමුම් රළ {height} m",
    swellPeriod: "ඉදිමුම් රළ {height} m, {period} s",
    swellFrom: "{direction} දෙසින් ඉදිමුම් රළ {height} m",
    swellPeriodFrom: "{direction} දෙසින් ඉදිමුම් රළ {height} m, {period} s",
    seaNote:
      "මුහුදු අගයන් රළ ආකෘතියකින් ලබාගත් ඒවාය. කාලගුණ විද්‍යා දෙපාර්තමේන්තුවේ මුහුදු ප්‍රදේශ අනතුරු ඇඟවීම් අනුගමනය කරන්න.",
  },
  chip: {
    title: "කාලගුණ මධ්‍යස්ථානය",
    observed: "නිරීක්ෂිත: {temperature} °C",
    dailyMaximum: "නිරීක්ෂිත, දෛනික උපරිමය: {temperature} °C",
    ageStale: "{age} · යල් පැන ගිය",
  },
  layer: {
    observed: "නිරීක්ෂිත",
    count: "මධ්‍යස්ථාන {count}",
    legendText:
      "කාලගුණ මධ්‍යස්ථානවල මනින ලදී, උෂ්ණත්වය අනුව වර්ණ ගන්වා ඇත. අවපැහැ කොටු යල් පැන ගිය ඒවාය.",
    legendTitle: "උෂ්ණත්වය",
    legendTitleObserved: "උෂ්ණත්වය, නිරීක්ෂිත",
  },
}
