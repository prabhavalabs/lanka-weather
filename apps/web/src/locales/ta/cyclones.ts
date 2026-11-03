import type { Translation } from "@/lib/i18n"
import type { cyclones as en } from "../en/cyclones"

export const cyclones: Translation<typeof en> = {
  title: "சூறாவளிகள்",
  cyclone: "சூறாவளி",
  newTab: "(புதிய தாவலில் திறக்கும்)",
  alert: "GDACS {level} விழிப்பூட்டல்",
  summary: {
    one: "{name} செயலில் உள்ளது",
    many: "செயலில் {count}: {names}",
  },
  listItem: "{name} · {distance}",
  checked: "சரிபார்க்கப்பட்டது {age}",
  missing: "இந்தத் தொகுதி இனி செயலில் உள்ளதாகப் பட்டியலிடப்படவில்லை.",
  status: {
    waiting: "சூறாவளித் தரவு இன்னும் சேகரிக்கப்படவில்லை.",
    activeStale:
      "அண்மையில் GDACS ஐச் சரிபார்க்க எம்மால் முடியவில்லை, எனவே இது காலாவதியாக இருக்கலாம்.",
    unknown:
      "அண்மையில் GDACS ஐச் சரிபார்க்க எம்மால் முடியவில்லை, எனவே சூறாவளி ஏதேனும் செயலில் உள்ளதா என்று கூற முடியாது. வளிமண்டலவியல் திணைக்களத்திடம் உறுதிப்படுத்துங்கள்.",
    none: "வட இந்து சமுத்திரத்தில் செயலில் உள்ள சூறாவளி இல்லை",
  },
  evaluation:
    "JTWC மற்றும் IMD RSMC புது டெல்லி அறிவுறுத்தல்கள் நேரடி மூலங்களாக இன்னும் மதிப்பீட்டில் உள்ளன. இங்குள்ள பாதைத் தரவு GDACS இனுடையது; அது JTWC ஐ அடிப்படையாகக் கொண்டது.",
  legend: {
    track: "இதுவரையான பாதை",
    forecast: "எதிர்வுகூறல்",
    buffer: "காற்று வலயம்",
    uncertainty: "நிச்சயமின்மை",
  },
  panel: {
    subtitle: "அயனமண்டலச் சூறாவளி · வட இந்து சமுத்திரம்",
    close: "சூறாவளி விவரங்களை மூடு",
    footer:
      "மூலம்: Global Disaster Alert and Coordination System (GDACS), gdacs.org. பெறப்பட்டது {age}.",
    fromCoast: "இலங்கைக் கரையிலிருந்து",
    maxWind: "அதிகபட்சக் காற்று",
    position: "அமைவிடம்",
    positionValue: "{lat}° {ns}, {lon}° {ew}",
    advisory: "அறிவுறுத்தல்",
    firstSeen: "முதலில் காணப்பட்டது",
    lists: "GDACS பட்டியலிடும் நாடுகள்",
    trackFrom: "பாதை மூலம்",
    report: "GDACS அறிக்கை",
    disclaimer:
      "GDACS விழிப்பூட்டல் அது பட்டியலிடும் நாடுகளில் ஏற்படக்கூடிய தாக்கத்தை மதிப்பிடுகிறது; அது இலங்கைக்கான எச்சரிக்கை அல்ல. உத்தியோகபூர்வ எச்சரிக்கைகள் வளிமண்டலவியல் திணைக்களம் மற்றும் DMC இடமிருந்து வருகின்றன.",
  },
}
