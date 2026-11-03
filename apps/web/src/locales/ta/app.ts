import type { Translation } from "@/lib/i18n"
import type { app as en } from "../en/app"

export const app: Translation<typeof en> = {
  shell: {
    tagline: "இலங்கையின் வானிலை மற்றும் அனர்த்த வரைபடம்",
    sidebarLabel: "அடுக்குகளும் அமைப்புகளும்",
    showLayers: "அடுக்குகளைக் காட்டு",
    hideLayers: "அடுக்குகளை மறை",
    layers: "அடுக்குகள்",
    layersOn: "அடுக்குகள், {count} இயக்கத்தில்",
    layersHint: "வரைபடம் எதைக் காட்ட வேண்டும் எனத் தெரிவுசெய்யுங்கள்.",
  },
  theme: {
    light: "வெளிர்",
    dark: "இருள்",
    system: "சாதனத்தின் அமைப்பு",
    auto: "தானியங்கி",
  },
  closeDetails: "விவரங்களை மூடு",
  mapLabel: "இலங்கை வரைபடம்",
}
