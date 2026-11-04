import type { Translation } from "@/lib/i18n"
import type { cyclones as en } from "../en/cyclones"

export const cyclones: Translation<typeof en> = {
  title: "සුළි කුණාටු",
  cyclone: "සුළි කුණාටුව",
  newTab: "(නව ටැබයක විවෘත වේ)",
  alert: "GDACS {level} ඇඟවීම",
  summary: {
    one: "{name} ක්‍රියාත්මකයි",
    many: "ක්‍රියාත්මක {count}: {names}",
  },
  listItem: "{name} · {distance}",
  checked: "පරීක්ෂා කළේ {age}",
  missing: "මෙම පද්ධතිය තවදුරටත් ක්‍රියාත්මක ලෙස ලැයිස්තුගත කර නැත.",
  status: {
    waiting: "සුළි කුණාටු දත්ත තවම එකතු කර නැත.",
    activeStale: "මෑතකදී GDACS පරීක්ෂා කිරීමට අපට නොහැකි වූ බැවින් මෙය යල් පැන ගොස් තිබිය හැක.",
    unknown:
      "මෑතකදී GDACS පරීක්ෂා කිරීමට අපට නොහැකි වූ බැවින් සුළි කුණාටුවක් ක්‍රියාත්මකදැයි කිව නොහැක. කාලගුණ විද්‍යා දෙපාර්තමේන්තුවෙන් විමසන්න.",
    none: "උතුරු ඉන්දියන් සාගරයේ ක්‍රියාත්මක සුළි කුණාටුවක් නැත",
  },
  evaluation:
    "JTWC සහ IMD RSMC නවදිල්ලි උපදේශන සෘජු මූලාශ්‍ර ලෙස තවමත් ඇගයීමට ලක් වෙමින් පවතී. මෙහි ඇති ගමන් මඟ දත්ත GDACS හි ය; එය JTWC මත පදනම් වේ.",
  legend: {
    track: "මෙතෙක් ගමන් මඟ",
    forecast: "පුරෝකථනය",
    buffer: "සුළං කලාපය",
    uncertainty: "අවිනිශ්චිතතාව",
  },
  panel: {
    subtitle: "නිවර්තන සුළි කුණාටුව · උතුරු ඉන්දියන් සාගරය",
    close: "සුළි කුණාටු විස්තර වසන්න",
    footer:
      "මූලාශ්‍රය: Global Disaster Alert and Coordination System (GDACS), gdacs.org. ලබා ගත්තේ {age}.",
    fromCoast: "ශ්‍රී ලංකා වෙරළේ සිට",
    maxWind: "උපරිම සුළං වේගය",
    position: "පිහිටීම",
    positionValue: "{lat}° {ns}, {lon}° {ew}",
    advisory: "උපදේශනය",
    firstSeen: "මුලින්ම දුටුවේ",
    lists: "GDACS ලැයිස්තුගත කරන රටවල්",
    trackFrom: "ගමන් මඟ ලබා ගත්තේ",
    report: "GDACS වාර්තාව",
    disclaimer:
      "GDACS ඇඟවීම එය ලැයිස්තුගත කරන රටවලට ඇති විය හැකි බලපෑම පිළිබඳ ඇගයීමකි; එය ශ්‍රී ලංකාව සඳහා අනතුරු ඇඟවීමක් නොවේ. නිල අනතුරු ඇඟවීම් නිකුත් කරන්නේ කාලගුණ විද්‍යා දෙපාර්තමේන්තුව සහ DMC ය.",
  },
}
