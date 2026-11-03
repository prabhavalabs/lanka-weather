import type { Translation } from "@/lib/i18n"
import type { search as en } from "../en/search"

export const search: Translation<typeof en> = {
  kind: {
    city: "නගරය",
    town: "කුඩා නගරය",
    suburb: "උප නගරය",
    village: "ගම",
    quarter: "නගර කොටස",
    neighbourhood: "පෙදෙස",
    hamlet: "කුඩා ගම්මානය",
    locality: "ප්‍රදේශය",
    isolated_dwelling: "හුදෙකලා නිවාසය",
    place: "ස්ථානය",
  },
  regionName: {
    province: "{name} පළාත",
    district: "{name} දිස්ත්‍රික්කය",
  },
  subtitle: "{kind} · {context}",
  station: "{basin} · ගංගා මිනුම් ස්ථානය",
  river: "ගඟ",
  tributary: "අතු ගඟ",
  reservoirNotMapped: "සිතියමේ නැත",
}
