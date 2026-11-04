import type { Translation } from "@/lib/i18n"
import type { grid as en } from "../en/grid"

export const grid: Translation<typeof en> = {
  variable: {
    temperature: { label: "උෂ්ණත්වය", short: "උෂ්ණත්වය" },
    precipitation: { label: "මෙම පැයේ වර්ෂාපතනය", short: "වැසි" },
    humidity: { label: "සාපේක්ෂ ආර්ද්‍රතාව", short: "ආර්ද්‍රතාව" },
    cloud: { label: "වලාකුළු ආවරණය", short: "වලාකුළු" },
    wind: { label: "සුළඟේ වේගය", short: "සුළඟ" },
  },
  legend: {
    temperature: "දැන් උෂ්ණත්වය",
    precipitation: "මෙම පැයේ වර්ෂාපතනය",
    light: "සිහින්",
    heavy: "තද",
    violent: "ඉතා තද",
    humidity: "සාපේක්ෂ ආර්ද්‍රතාව",
    humidityLow: "වැලි පැහැය",
    humidityHigh: "නිල්",
    cloud: "වලාකුළු ආවරණය",
    cloudLow: "පැහැදිලි",
    cloudHigh: "වලාකුළින් වැසුණු",
    wind: "දැන් සුළඟේ වේගය",
  },
  wind: {
    calm: "සුළං නැති තරම්",
    text: "{direction} දෙසින් {kmh} km/h ({knots} kt)",
  },
  timeline: {
    section: "සුළං පුරෝකථන වේලාව",
    play: "සුළං පුරෝකථනය ධාවනය කරන්න",
    pause: "සුළං පුරෝකථනය නවත්වන්න",
    thumb: "පුරෝකථන වේලාව",
    thumbValue: "{time}, {lead}",
    scale: "සුළඟේ වේගය අනුව අංශුවල වර්ණ, km/h",
    source: "පොළොවේ සිට 10 m සුළඟ · NOAA GFS",
    sourceLoading: "පොළොවේ සිට 10 m සුළඟ · NOAA GFS · පූරණය වෙමින්…",
    run: "ආකෘති ධාවනය {time}",
    now: "දැන්",
    leadHours: "+පැය {hours}",
    leadDays: "+දින {days}",
    leadDaysHours: "+දින {days} පැය {hours}",
  },
}
