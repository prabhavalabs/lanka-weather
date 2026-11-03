import type { Translation } from "@/lib/i18n"
import type { rain as en } from "../en/rain"

export const rain: Translation<typeof en> = {
  step: "{day} {start}–{end}",
  method: {
    dailyReport: "කාලගුණ විද්‍යා දෙපාර්තමේන්තුවේ දෛනික වාර්තාව, පෙ.ව. 08:30 දක්වා පැය 24",
    sum3h: "කාලගුණ විද්‍යා දෙපාර්තමේන්තුවේ පැය 3 කියවීම්වල එකතුව",
    sumGauge: "පැය පතා ගංගා ජල මාපක කියවීම්වල එකතුව",
  },
  timeline: {
    section: "චන්ද්‍රිකා වර්ෂා රූපයේ වේලාව",
    play: "වර්ෂා රූප ධාවනය කරන්න",
    pause: "වර්ෂා රූප නවත්වන්න",
    thumb: "චන්ද්‍රිකා වර්ෂා රූපයේ වේලාව",
    thumbValue: "{time}, {age}",
    note: "ඇස්තමේන්තුව · NASA IMERG · පැය කිහිපයක් ප්‍රමාදයි",
    outOfDate: "යල් පැන ගිය",
    scale: "චන්ද්‍රිකා රූපයේ වර්ණ, {from} සිට {to} {unit} සහ ඊට වැඩි",
  },
  layer: {
    satellite: "චන්ද්‍රිකා ඇස්තමේන්තුව",
    points: "මනින ලද ලක්ෂ්‍ය {count}",
    satelliteText: "NASA IMERG, මිනුමක් නොවේ. රූප පැය කිහිපයක් ප්‍රමාද වේ.",
    satelliteTextNewest:
      "NASA IMERG, මිනුමක් නොවේ. රූප පැය කිහිපයක් ප්‍රමාද වේ; නවතම රූපය ලැබුණේ {age}.",
    measuredTitle: "මනින ලද, පසුගිය පැය 24",
    measuredText:
      "වෘත්ත ගංගා ජල මාපක ද, කොටු කාලගුණ මධ්‍යස්ථාන ද වේ. හිස් ලක්ෂ්‍ය වර්තමාන ලෙස සැලකීමට තරම් පැරණිය; දිනය බැලීමට එකක් විවෘත කරන්න. සජීවී වර්ෂාමානයක් අසල ඇති විට, එම ස්තරය ක්‍රියාත්මක අතරතුර සජීවී වර්ෂාමානය පමණක් පෙන්වයි.",
    measuredCounts: "වර්තමාන {fresh}, පැරණි {stale}.",
    forecastTitle: "පුරෝකථනය",
    forecastText: "ඉදිරි වර්ෂාව කාලගුණ ක්ෂේත්‍රයේ සහ ස්ථාන පැනලයේ ඇත. එය ආකෘති දත්තයකි.",
    showForecast: "පුරෝකථන වර්ෂාව පෙන්වන්න",
  },
  gauges: {
    band: {
      veryHeavy: "ඉතා තද වැසි",
      heavy: "තද වැසි",
      moderate: "මධ්‍යස්ථ වැසි",
      light: "සුළු වැසි",
      none: "වැසි නැත",
      unknown: "යල් පැන ගිය",
    },
    range: {
      veryHeavy: "100 mm හෝ ඊට වැඩි",
      heavy: "50 සිට 100 mm",
      moderate: "10 සිට 50 mm",
      light: "10 mm ට අඩු",
      none: "0 mm",
      unknown: "නවතම කියවීම පැය 3කට වඩා පැරණියි",
    },
    summary: "{total} න් {fresh} ක් වර්තමානයි · {wet} ක 10 mm හෝ ඊට වැඩි",
    legendTitle: "වර්ෂාමාන",
    legendUnit: "පසුගිය පැය 24",
    layerText:
      "මනින ලද, පැය 24 චලිත එකතු. සුළු වැසි සඳහා, පසුව වියළි වර්ෂාමාන සඳහා විශාලනය කරන්න; හිස් බිංදුවක් යනු පැය 3කට වැඩි කාලයක් වාර්තා නොකළ වර්ෂාමානයකි, එයින් වියළි බව අදහස් නොවේ.",
  },
  panel: {
    title: "වර්ෂාමානය",
    subtitle: "වර්ෂාමානය · {institution}",
    close: "වර්ෂාමානය වසන්න",
    error: "මෙම වර්ෂාමානය මේ මොහොතේ කියවිය නොහැකි විය.",
    unit: "mm, පැය 24 තුළ",
    silent:
      "මෙම වර්ෂාමානය පැය 3කට වැඩි කාලයක් වාර්තා කර නැති නිසා, වැසි කොපමණ ඇද හැලුණේදැයි කිව නොහැක. එයින් වියළි කාලගුණයක් පැවති බව අදහස් නොවේ.",
    lastFigure: "එහි අවසන් අගය {rain} mm, {time}.",
    measured: "මනින ලද",
    read: "කියවූ වේලාව",
    readValue: "{time} · {age}",
    noReading: "කියවීමක් නැත",
    runBy: "පවත්වාගෙන යන්නේ",
    runByNamed: "පවත්වාගෙන යන්නේ {institution}.",
    rolling: "පැය 24 චලිත එකතුව, පැය 48 දක්වා",
    rollingNote:
      "සෑම ලක්ෂ්‍යයක්ම ඊට පෙර පැය 24 හි එකතුවයි, එබැවින් එය කිසියම් එක් පැයක් ගැන කිසිවක් නොකියයි.",
    chartLabel: "පැය 24 චලිත වර්ෂාව {low} සිට {peak} mm දක්වා",
    upTo: "{peak} mm දක්වා",
  },
  chip: {
    gauge: "වර්ෂාමානය",
    rainfall: "වර්ෂාපතනය",
    total: "වර්ෂාව, පැය 24: {rain}",
    totalBand: "වර්ෂාව, පැය 24: {rain} · {band}",
    noTotal: "යල් පැන ගිය: වර්තමාන එකතුවක් නැත",
    runBy: "{institution} · {age}",
    ending: "{age} අවසන් විය",
    endingStale: "{age} අවසන් විය · යල් පැන ගිය",
  },
}
