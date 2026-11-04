/** The languages of Sri Lanka the interface is written in. */
export type Locale = "en" | "si" | "ta"

export const localeOrder: readonly Locale[] = ["si", "ta", "en"]

/** Must match the inline script in index.html, which sets `lang` before first paint. */
export const localeStorageKey = "lanka-weather:locale"

export const locales: Record<
  Locale,
  {
    /** BCP 47 tag for `lang` and `Intl`. */
    tag: string
    /** The language's name in itself, for the switcher. */
    nativeName: string
    /** The language's name in English. */
    englishName: string
  }
> = {
  si: { tag: "si-LK", nativeName: "සිංහල", englishName: "Sinhala" },
  ta: { tag: "ta-LK", nativeName: "தமிழ்", englishName: "Tamil" },
  en: { tag: "en-GB", nativeName: "English", englishName: "English" },
}

export function isLocale(value: unknown): value is Locale {
  return value === "en" || value === "si" || value === "ta"
}
