import type { Translation } from "@/lib/i18n"
import type { search as en } from "../en/search"

export const search: Translation<typeof en> = {
  kind: {
    city: "நகரம்",
    town: "பட்டினம்",
    suburb: "புறநகர்",
    village: "கிராமம்",
    quarter: "நகரப் பகுதி",
    neighbourhood: "அயல் பகுதி",
    hamlet: "சிறு கிராமம்",
    locality: "வட்டாரம்",
    isolated_dwelling: "தனி வீடு",
    place: "இடம்",
  },
  regionName: {
    province: "{name} மாகாணம்",
    district: "{name} மாவட்டம்",
  },
  subtitle: "{kind} · {context}",
  station: "{basin} · ஆற்று அளவீட்டு நிலையம்",
  river: "ஆறு",
  tributary: "கிளை ஆறு",
  reservoirNotMapped: "வரைபடத்தில் இல்லை",
}
