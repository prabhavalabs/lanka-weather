import type { Translation } from "@/lib/i18n"
import type { floodRisk as en } from "../en/floodRisk"

export const floodRisk: Translation<typeof en> = {
  title: "ගංවතුර අවදානම් ඇස්තමේන්තුව",
  level: {
    none: "ඒකාබද්ධ සංඥාවක් නැත",
    moderate: "මධ්‍යම",
    high: "ඉහළ",
    very_high: "ඉතා ඉහළ",
  },
  none: "දැන් මෙහි ගංවතුරක් ගැන මූලාශ්‍ර දෙකක් එකඟ නොවේ. එය ආරක්ෂිත බවට ප්‍රකාශයක් නොවේ.",
  scenario: {
    flash: "පැය කිහිපයක් ඇතුළත පහත් බිම් සහ මාර්ග යට විය හැක.",
    river: "ගංගා පිටාර ගලා ඉවුරු අසල පහත් බිම් යට විය හැක.",
    both: "තද වැසි සහ ඉහළ යන ගංගා නිසා පහත් බිම් යට විය හැක.",
    general: "පහත් බිම් යට විය හැක.",
  },
  signal: {
    official: "නිල නිවේදනය",
    google_flash: "Google හදිසි ගංවතුර පුරෝකථනය",
    rain_now: "මනින ලද වැසි",
    river: "ගංගා මිනුම් ස්ථාන",
    google_gauge: "Google ගංගා පුරෝකථනය",
    rain_forecast: "වැසි පුරෝකථනය",
    wet_ground: "දැනටමත් ලැබුණු වැසි",
    reservoir: "ජලාශ",
  },
  value: {
    official: { advisory: "උපදේශනය", warning: "අනතුරු ඇඟවීම", red: "රතු අනතුරු ඇඟවීම" },
    flash: { likely: "ඇති විය හැකිය", highly_likely: "ඇති වීමට බොහෝ ඉඩ ඇත" },
    gauge: { warning: "අනතුරු ඇඟවීම", danger: "අනතුර", extreme: "දැඩි අනතුර" },
    river: {
      alert: "අනතුරු ඇඟවීමේ මට්ටම",
      minor_flood: "සුළු ගංවතුර මට්ටම",
      major_flood: "මහා ගංවතුර මට්ටම",
    },
    rain1h: "පැයක් තුළ අවම වශයෙන් මි.මී. {mm}",
    rain3h: "පැය 3ක් තුළ අවම වශයෙන් මි.මී. {mm}",
    prior: "ඊට පෙර දිනය තුළ මි.මී. {mm}",
    forecast6h: "ඉදිරි පැය 6 තුළ මි.මී. {mm}",
    forecast24h: "ඉදිරි පැය 24 තුළ මි.මී. {mm}",
    reservoir: { one: "{count}ක් වාන් දමයි", other: "{count}ක් වාන් දමයි" },
  },
  note: "අපගේ ඇස්තමේන්තුවකි, නිල අනතුරු ඇඟවීමක් නොවේ. DMC උපදෙස් පිළිපදින්න.",
  incomplete: "සමහර මූලාශ්‍ර කියවිය නොහැකි විය, එබැවින් එය අඩු විය හැක.",
}
