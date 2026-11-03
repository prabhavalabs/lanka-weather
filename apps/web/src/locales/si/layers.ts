import type { Translation } from "@/lib/i18n"
import type { layers as en } from "../en/layers"

export const layers: Translation<typeof en> = {
  category: {
    hazard: "උපද්‍රව සහ අනතුරු ඇඟවීම්",
    hydrology: "ගංගා සහ ගංවතුර",
    weather: "කාලගුණය",
    geography: "භූගෝලය",
  },
  status: {
    waiting: "දත්ත තවම ලැබෙමින් පවතී · ඉක්මනින් එක්රැස් කෙරේ",
    error: "ලබාගත නොහැක · ඉක්මනින් නැවත උත්සාහ කෙරේ",
  },
  age: {
    updated: "{age} යාවත්කාලීන විය",
    read: "{age} කියවිණි",
    run: "{age} ධාවනය",
    image: "රූපය {age}",
    newest: "නවතම {age}",
    issued: "{age} නිකුත් විය",
  },
}
