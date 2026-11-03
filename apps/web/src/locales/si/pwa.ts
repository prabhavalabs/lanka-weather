import type { Translation } from "@/lib/i18n"
import type { pwa as en } from "../en/pwa"

export const pwa: Translation<typeof en> = {
  update: {
    title: "නව සංස්කරණයක් සූදානම්",
    description: "එය භාවිත කිරීමට නැවත පූරණය කරන්න. ඔබ විවෘත කර ඇති කිසිවක් නැති නොවේ.",
    reload: "නැවත පූරණය",
    later: "පසුව",
  },
  offline: {
    title: "ඔබ නොබැඳි ය",
    body: "සිතියම අවසන් වරට සුරැකි දත්ත පෙන්වයි. සෑම ස්තරයක්ම එය කොතරම් පැරණි දැයි පෙන්වයි.",
    back: "නැවත සම්බන්ධ විය. සිතියම යාවත්කාලීන වෙමින් පවතී.",
  },
  ready: "Lanka Weather දැන් අන්තර්ජාලය නැතිව ද ක්‍රියා කරයි.",
  push: {
    genericTitle: "Lanka Weather",
    genericBody: "නව යාවත්කාලීනයක් ඇත. එය බැලීමට සිතියම විවෘත කරන්න.",
  },
}
