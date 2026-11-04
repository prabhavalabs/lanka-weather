import type { Translation } from "@/lib/i18n"
import type { regions as en } from "../en/regions"

export const regions: Translation<typeof en> = {
  variable: {
    temperature: { label: "தற்போதைய வெப்பநிலை", short: "வெப்பநிலை" },
    rain: { label: "இன்றைய மழை", short: "மழை" },
    wind: { label: "தற்போதைய அதிகூடிய காற்று", short: "காற்று" },
  },
  by: {
    province: "மாகாண வாரியாக",
    district: "மாவட்ட வாரியாக",
    ds: "பி.செ. பிரிவு வாரியாக",
    gn: "கிராம உத்தியோகத்தர் பிரிவு வாரியாக",
  },
  legend: {
    temperature: "{by} வெப்பநிலை",
    rain: "{by} இன்றைய மழை",
    wind: "{by} அதிகூடிய காற்று",
  },
  aggregate: {
    title: "{level} முழுவதும்",
    meanNow: "தற்போதைய சராசரி",
    today: "இன்று",
    maxWind: "அதிகூடிய காற்று",
    rainNow: "தற்போதைய மழை",
    rainToday: "இன்றைய மழை",
    maxGusts: "அதிகூடிய காற்று வீச்சு",
    cells: {
      one: "{count} வலைக் கட்டத்தின் சராசரி, பரப்பளவுக்கேற்ப நிறையிடப்பட்டது.",
      other: "{count} வலைக் கட்டங்களின் சராசரி, பரப்பளவுக்கேற்ப நிறையிடப்பட்டது.",
    },
    point: "பிரதேசத்தினுள் ஒரு புள்ளியில் பெறப்பட்ட அளவு.",
  },
}
