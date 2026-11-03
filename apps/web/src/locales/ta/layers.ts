import type { Translation } from "@/lib/i18n"
import type { layers as en } from "../en/layers"

export const layers: Translation<typeof en> = {
  category: {
    hazard: "அனர்த்தங்களும் எச்சரிக்கைகளும்",
    hydrology: "ஆறுகளும் வெள்ளமும்",
    weather: "வானிலை",
    geography: "புவியியல்",
  },
  status: {
    waiting: "தரவு இன்னும் வருகிறது · விரைவில் சேகரிக்கப்படும்",
    error: "கிடைக்கவில்லை · விரைவில் மீண்டும் முயலப்படும்",
  },
  age: {
    updated: "{age} புதுப்பிக்கப்பட்டது",
    read: "{age} அளவிடப்பட்டது",
    run: "{age} இயக்கப்பட்டது",
    image: "படம் {age}",
    newest: "புதியது {age}",
    issued: "{age} வெளியிடப்பட்டது",
  },
}
