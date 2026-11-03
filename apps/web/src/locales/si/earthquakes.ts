import type { Translation } from "@/lib/i18n"
import type { earthquakes as en } from "../en/earthquakes"

export const earthquakes: Translation<typeof en> = {
  title: "භූකම්පන",
  earthquake: "භූකම්පනය",
  newTab: "(නව ටැබයක විවෘත වේ)",
  hoverTitle: "M{magnitude} භූකම්පනය",
  listTitle: "M{magnitude} {place}",
  fromSriLanka: "ශ්‍රී ලංකාවේ සිට {distance}",
  missing: "මෙම සිදුවීම USGS ලැයිස්තුවේ පසුගිය දින 30 තුළ නැත. එය සංශෝධනය කර හෝ ඉවත් කර තිබිය හැක.",
  age: {
    day: "පසුගිය පැය 24",
    week: "පසුගිය දින 7",
    month: "පසුගිය දින 30",
  },
  summary: {
    none: "දින {days} තුළ M4.5+ කිසිවක් නැත",
    count: "දින {days} තුළ {count}",
    strongest: "ප්‍රබලතම M{magnitude}",
  },
  tsunami:
    "USGS මෙම සිදුවීමට සුනාමි සලකුණ යොදා ඇත. එම සලකුණෙන් පෙන්වන්නේ සිදුවීම සාගර කලාපයක සිදු වූ, සලකා බැලිය යුතු තරම් විශාල එකක් බව පමණි; එය ශ්‍රී ලංකාව සඳහා අනතුරු ඇඟවීමක් නොවේ. සුනාමි අනතුරු ඇඟවීම් සඳහා ආපදා කළමනාකරණ මධ්‍යස්ථානය සහ කාලගුණ විද්‍යා දෙපාර්තමේන්තුව අනුගමනය කරන්න.",
  status: {
    reviewed: "සමාලෝචිත",
    automatic: "ස්වයංක්‍රීය",
    deleted: "ඉවත් කළ",
  },
  legend: {
    size: "කවයේ ප්‍රමාණයෙන් ප්‍රබලත්වය දැක්වේ",
    tsunami: "USGS සුනාමි සලකුණ, අනතුරු ඇඟවීමක් නොවේ",
  },
  panel: {
    subtitle: "භූකම්පනය · USGS",
    close: "භූකම්පන විස්තර වසන්න",
    footer: "මූලාශ්‍රය: U.S. Geological Survey. සංශෝධනය කළේ {revised}, ලබා ගත්තේ {fetched}.",
    magnitude: "ප්‍රබලත්වය",
    time: "වේලාව",
    depth: "ගැඹුර",
    fromCoast: "ශ්‍රී ලංකා වෙරළේ සිට",
    pager: "USGS PAGER ඇඟවීම",
    eventPage: "USGS සිදුවීම් පිටුව",
    noWarnings:
      "Lanka Weather සුනාමි අනතුරු ඇඟවීම් නිකුත් නොකරයි. නිල අනතුරු ඇඟවීම් නිකුත් කරන්නේ DMC සහ කාලගුණ විද්‍යා දෙපාර්තමේන්තුවයි.",
  },
  details: {
    latest: "නවතම",
    showOcean: "ඉන්දියන් සාගරය පෙන්වන්න",
    coverage:
      "දකුණු අක්ෂාංශ 15° සිට උතුරු අක්ෂාංශ 25° දක්වා සහ නැගෙනහිර දේශාංශ 60° සිට 105° දක්වා M4.5 සහ ඊට වැඩි භූකම්පන, සහ ශ්‍රී ලංකාව අසල USGS ලැයිස්තුගත කරන කුඩා භූකම්පන. Lanka Weather සුනාමි අනතුරු ඇඟවීම් නිකුත් නොකරයි.",
  },
}
