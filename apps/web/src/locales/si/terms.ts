import type { Translation } from "@/lib/i18n"
import type { terms as en } from "../en/terms"

export const terms: Translation<typeof en> = {
  riverStatus: {
    normal: "සාමාන්‍ය",
    alert: "අනතුරු ඇඟවීම",
    minor_flood: "සුළු ගංවතුර",
    major_flood: "ප්‍රධාන ගංවතුර",
    unknown: "මට්ටම් නොදනී",
  },
  riverLevel: {
    alert: "අනතුරු ඇඟවීමේ මට්ටම",
    minor_flood: "සුළු ගංවතුර මට්ටම",
    major_flood: "ප්‍රධාන ගංවතුර මට්ටම",
  },
  trend: {
    rising: "ඉහළ යමින්",
    falling: "පහළ බසිමින්",
    steady: "වෙනසක් නැත",
    unknown: "ප්‍රවණතාව නොදනී",
  },
  landslideLevel: {
    l1: "1 වන මට්ටම (කහ)",
    l2: "2 වන මට්ටම (තැඹිලි)",
    l3: "3 වන මට්ටම (රතු)",
  },
  landslideAction: {
    l1: "අවධානයෙන් සිටින්න",
    l2: "සූදානම් වන්න",
    l3: "ඉවත් වන්න",
  },
  colour: {
    red: "රතු",
    amber: "තැඹිලි",
    yellow: "කහ",
    orange: "තැඹිලි",
    green: "කොළ",
  },
  bulletinSeverity: {
    warning: "අනතුරු ඇඟවීම",
    advisory: "උපදේශනය",
    withdrawal: "ඉවත් කිරීම",
    routine: "වාර්තාව",
  },
  bulletinType: {
    landslide: "නායයෑම්",
    water_level: "ජල මට්ටම",
    weather: "කාලගුණය",
    situation: "තත්ත්වය",
  },
  weatherSystem: {
    lowPressure: "අඩු පීඩන කලාපය",
    depression: "අවපාතය",
    deepDepression: "ගැඹුරු අවපාතය",
    cyclonicStorm: "සුළි කුණාටුව",
    severeCyclonicStorm: "දරුණු සුළි කුණාටුව",
    cyclone: "සුළි සුළඟ",
  },
  reservoirStatus: {
    spilling: "වාන් දමමින්",
    releasing: "ජලය මුදාහරිමින්",
    holding: "ජලය රඳවා ඇත",
    nodata: "දත්ත නැත",
  },
  adminLevel: {
    province: "පළාත",
    district: "දිස්ත්‍රික්කය",
    ds: "ප්‍රා.ලේ. කොට්ඨාසය",
    gn: "ග්‍රා.නි. වසම",
    dsLong: "ප්‍රාදේශීය ලේකම් කොට්ඨාසය",
    gnLong: "ග්‍රාම නිලධාරී වසම",
  },
  adminCount: {
    province: "පළාත් {count}",
    district: "දිස්ත්‍රික්ක {count}",
    ds: "ප්‍රා.ලේ. කොට්ඨාස {count}",
    gn: "ග්‍රා.නි. වසම් {count}",
  },
  agency: {
    dmc: "ආපදා කළමනාකරණ මධ්‍යස්ථානය",
    nbro: "ජාතික ගොඩනැගිලි පර්යේෂණ සංවිධානය",
    meteo: "කාලගුණ විද්‍යා දෙපාර්තමේන්තුව",
    irrigation: "වාරිමාර්ග දෙපාර්තමේන්තුව",
    earlyWarning: "ස්වාභාවික උපද්‍රව පූර්ව අනතුරු ඇඟවීමේ මධ්‍යස්ථානය",
  },
  estimate: "ඇස්තමේන්තුව",
  estimated: "ඇස්තමේන්තුගත",
  official: "නිල",
  officialWarning: "නිල අනතුරු ඇඟවීම",
  notOfficial: "මෙය නිල අනතුරු ඇඟවීමක් නොවේ.",
  emergency: "හදිසි අවස්ථාවකදී 117 අමතන්න.",
}
