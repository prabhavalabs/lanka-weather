import { describe, expect, it } from "vitest"
import { translateIn, type MessageKey } from "@/lib/i18n"
import { install } from "@/locales/en/install"
import { pwa } from "@/locales/en/pwa"
import { guides } from "./guides"
import { guideIds } from "./platform"

/** Paths of objects the language layer would read as a plural message (a string `other`). */
function pluralShaped(node: object, path: string[] = []): string[] {
  return Object.entries(node).flatMap(([key, value]) => {
    if (typeof value !== "object" || value === null) return []
    const here = [...path, key]
    const self = typeof (value as { other?: unknown }).other === "string" ? [here.join(".")] : []
    return [...self, ...pluralShaped(value, here)]
  })
}

describe("install and pwa catalogues", () => {
  it("have no group the language layer would take for a plural", () => {
    expect(pluralShaped({ install, pwa })).toEqual([])
  })

  it("give every step and note of every guide a message in each language", () => {
    for (const id of guideIds) {
      const keys: MessageKey[] = guides[id].steps.map((step) => step.text)
      if (guides[id].note) keys.push(guides[id].note)
      for (const key of keys)
        for (const locale of ["en", "si", "ta"] as const)
          expect(translateIn(locale, key), `${locale} ${key}`).not.toBe(key)
    }
  })
})
