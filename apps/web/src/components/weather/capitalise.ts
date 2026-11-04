import { intlTag } from "@/lib/i18n"

/** The first letter in upper case, for a word used as a label ("today" → "Today"); no-op in Sinhala and Tamil. */
export function capitalise(text: string): string {
  return text.charAt(0).toLocaleUpperCase(intlTag()) + text.slice(1)
}
