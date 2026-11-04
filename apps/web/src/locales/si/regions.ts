import type { Translation } from "@/lib/i18n"
import type { regions as en } from "../en/regions"

export const regions: Translation<typeof en> = {
  variable: {
    temperature: { label: "දැන් උෂ්ණත්වය", short: "උෂ්ණත්වය" },
    rain: { label: "අද වර්ෂාපතනය", short: "වැසි" },
    wind: { label: "දැන් ප්‍රබලතම සුළඟ", short: "සුළඟ" },
  },
  by: {
    province: "පළාත් අනුව",
    district: "දිස්ත්‍රික්ක අනුව",
    ds: "ප්‍රා.ලේ. කොට්ඨාස අනුව",
    gn: "ග්‍රා.නි. වසම් අනුව",
  },
  legend: {
    temperature: "{by} උෂ්ණත්වය",
    rain: "{by} අද වර්ෂාපතනය",
    wind: "{by} උපරිම සුළඟ",
  },
  aggregate: {
    title: "{level} සමස්තයක් ලෙස",
    meanNow: "දැන් සාමාන්‍යය",
    today: "අද",
    maxWind: "උපරිම සුළඟ",
    rainNow: "දැන් වැසි",
    rainToday: "අද වැසි",
    maxGusts: "උපරිම සුළං රැල්ල",
    cells: "ජාල කොටු {count}ක සාමාන්‍යය, වර්ගඵලය අනුව බර තබා ඇත.",
    point: "කලාපය තුළ එක් ස්ථානයක අගය.",
  },
}
