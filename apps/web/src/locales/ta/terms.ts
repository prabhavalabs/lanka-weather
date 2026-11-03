import type { Translation } from "@/lib/i18n"
import type { terms as en } from "../en/terms"

export const terms: Translation<typeof en> = {
  riverStatus: {
    normal: "சாதாரண",
    alert: "எச்சரிக்கை",
    minor_flood: "சிறு வெள்ளம்",
    major_flood: "பெரு வெள்ளம்",
    unknown: "மட்டங்கள் இல்லை",
  },
  riverLevel: {
    alert: "எச்சரிக்கை மட்டம்",
    minor_flood: "சிறு வெள்ள மட்டம்",
    major_flood: "பெரு வெள்ள மட்டம்",
  },
  trend: {
    rising: "உயர்கிறது",
    falling: "குறைகிறது",
    steady: "மாற்றமில்லை",
    unknown: "போக்கு தெரியவில்லை",
  },
  landslideLevel: {
    l1: "நிலை 1 (மஞ்சள்)",
    l2: "நிலை 2 (செம்மஞ்சள்)",
    l3: "நிலை 3 (சிவப்பு)",
  },
  landslideAction: {
    l1: "அவதானமாக இருங்கள்",
    l2: "தயாராக இருங்கள்",
    l3: "வெளியேறுங்கள்",
  },
  colour: {
    red: "சிவப்பு",
    amber: "செம்மஞ்சள்",
    yellow: "மஞ்சள்",
    orange: "செம்மஞ்சள்",
    green: "பச்சை",
  },
  bulletinSeverity: {
    warning: "எச்சரிக்கை",
    advisory: "அறிவுறுத்தல்",
    withdrawal: "மீளப்பெறல்",
    routine: "அறிக்கை",
  },
  bulletinType: {
    landslide: "மண்சரிவு",
    water_level: "நீர் மட்டம்",
    weather: "வானிலை",
    situation: "நிலைமை",
  },
  weatherSystem: {
    lowPressure: "தாழமுக்கப் பிரதேசம்",
    depression: "தாழமுக்கம்",
    deepDepression: "ஆழமான தாழமுக்கம்",
    cyclonicStorm: "சூறாவளிப் புயல்",
    severeCyclonicStorm: "கடும் சூறாவளிப் புயல்",
    cyclone: "சூறாவளி",
  },
  reservoirStatus: {
    spilling: "வான் பாய்கிறது",
    releasing: "நீர் திறந்துவிடப்படுகிறது",
    holding: "நீர் தேக்கப்பட்டுள்ளது",
    nodata: "தரவு இல்லை",
  },
  adminLevel: {
    province: "மாகாணம்",
    district: "மாவட்டம்",
    ds: "பி.செ. பிரிவு",
    gn: "கி.அ. பிரிவு",
    dsLong: "பிரதேச செயலகப் பிரிவு",
    gnLong: "கிராம அலுவலர் பிரிவு",
  },
  adminCount: {
    province: "{count} மாகாணங்கள்",
    district: "{count} மாவட்டங்கள்",
    ds: "{count} பி.செ. பிரிவுகள்",
    gn: "{count} கி.அ. பிரிவுகள்",
  },
  agency: {
    dmc: "அனர்த்த முகாமைத்துவ நிலையம்",
    nbro: "தேசிய கட்டட ஆராய்ச்சி நிறுவனம்",
    meteo: "வளிமண்டலவியல் திணைக்களம்",
    irrigation: "நீர்ப்பாசனத் திணைக்களம்",
    earlyWarning: "இயற்கை அனர்த்த முன்னெச்சரிக்கை நிலையம்",
  },
  estimate: "மதிப்பீடு",
  estimated: "மதிப்பிடப்பட்ட",
  official: "உத்தியோகபூர்வ",
  officialWarning: "உத்தியோகபூர்வ எச்சரிக்கை",
  notOfficial: "இது உத்தியோகபூர்வ எச்சரிக்கை அல்ல.",
  emergency: "அவசர நிலையில் 117 ஐ அழைக்கவும்.",
}
