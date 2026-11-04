import type { Translation } from "@/lib/i18n"
import type { terms as en } from "../en/terms"

export const terms: Translation<typeof en> = {
  riverStatus: {
    normal: "சாதாரண",
    alert: "எச்சரிக்கை",
    minor_flood: "சிறிய வெள்ளம்",
    major_flood: "பாரிய வெள்ளம்",
    unknown: "மட்டங்கள் இல்லை",
  },
  riverLevel: {
    alert: "எச்சரிக்கை மட்டம்",
    minor_flood: "சிறிய வெள்ள மட்டம்",
    major_flood: "பாரிய வெள்ள மட்டம்",
  },
  trend: {
    rising: "உயர்ந்து வருகிறது",
    falling: "குறைந்து வருகிறது",
    steady: "மாற்றமில்லை",
    unknown: "போக்கு தெரியவில்லை",
  },
  landslideLevel: {
    l1: "எச்சரிக்கை மட்டம் 1 (மஞ்சள்)",
    l2: "எச்சரிக்கை மட்டம் 2 (செம்மஞ்சள்)",
    l3: "எச்சரிக்கை மட்டம் 3 (சிவப்பு)",
  },
  landslideAction: {
    l1: "அவதானிப்பு",
    l2: "எச்சரிக்கை",
    l3: "வெளியேறுகை",
  },
  colour: {
    red: "சிவப்பு",
    amber: "ஆம்பர்",
    yellow: "மஞ்சள்",
    orange: "செம்மஞ்சள்",
    green: "பச்சை",
  },
  bulletinSeverity: {
    warning: "எச்சரிக்கை",
    advisory: "ஆலோசனை",
    withdrawal: "விலக்கிக்கொள்ளல்",
    routine: "அறிக்கை",
  },
  bulletinType: {
    landslide: "மண்சரிவு",
    water_level: "நீர் மட்டம்",
    weather: "வானிலை",
    situation: "நிலைமை",
  },
  weatherSystem: {
    lowPressure: "குறைந்த அழுத்தப் பிரதேசம்",
    depression: "தாழமுக்கம்",
    deepDepression: "ஆழமான தாழமுக்கம்",
    cyclonicStorm: "புயல்",
    severeCyclonicStorm: "பாரிய புயல்",
    cyclone: "சூறாவளி",
  },
  reservoirStatus: {
    spilling: "வான் பாய்கிறது",
    releasing: "நீர் வெளியேற்றப்படுகிறது",
    holding: "நீர் தேக்கப்பட்டுள்ளது",
    nodata: "தரவு இல்லை",
  },
  adminLevel: {
    province: "மாகாணம்",
    district: "மாவட்டம்",
    ds: "பி.செ. பிரிவு",
    gn: "கி.உ. பிரிவு",
    dsLong: "பிரதேச செயலகப் பிரிவு",
    gnLong: "கிராம உத்தியோகத்தர் பிரிவு",
  },
  adminCount: {
    province: "{count} மாகாணங்கள்",
    district: "{count} மாவட்டங்கள்",
    ds: "{count} பி.செ. பிரிவுகள்",
    gn: "{count} கி.உ. பிரிவுகள்",
  },
  agency: {
    dmc: "அனர்த்த முகாமைத்துவ நிலையம்",
    nbro: "தேசிய கட்டிட ஆராய்ச்சி அமைப்பு",
    meteo: "வளிமண்டலவியல் திணைக்களம்",
    irrigation: "நீர்ப்பாசனத் திணைக்களம்",
    earlyWarning: "இயற்கை அனர்த்தம் தொடர்பான முன்னெச்சரிக்கை நிலையம்",
  },
  estimate: "மதிப்பீடு",
  estimated: "மதிப்பிடப்பட்ட",
  official: "உத்தியோகபூர்வ",
  officialWarning: "உத்தியோகபூர்வ எச்சரிக்கை",
  notOfficial: "இது உத்தியோகபூர்வ எச்சரிக்கை அல்ல.",
  emergency: "அவசர நிலையில் 117 ஐ அழைக்கவும்.",
}
