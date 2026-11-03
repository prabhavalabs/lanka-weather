import type { Translation } from "@/lib/i18n"
import type { app as en } from "../en/app"

export const app: Translation<typeof en> = {
  shell: {
    tagline: "ශ්‍රී ලංකාවේ කාලගුණ සහ උපද්‍රව සිතියම",
    sidebarLabel: "ස්තර සහ සැකසුම්",
    showLayers: "ස්තර පෙන්වන්න",
    hideLayers: "ස්තර සඟවන්න",
    layers: "ස්තර",
    layersOn: "ස්තර, {count}ක් ක්‍රියාත්මකයි",
    layersHint: "සිතියමේ පෙන්විය යුතු දේ තෝරන්න.",
  },
  theme: {
    light: "ආලෝකමත්",
    dark: "අඳුරු",
    system: "උපාංගයේ සැකසුම ලෙස",
    auto: "ස්වයං",
  },
  closeDetails: "විස්තර වසන්න",
  mapLabel: "ශ්‍රී ලංකා සිතියම",
}
