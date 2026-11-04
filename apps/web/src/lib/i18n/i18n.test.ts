import { describe, expect, it } from "vitest"
import { en } from "@/locales/en"
import { si } from "@/locales/si"
import { ta } from "@/locales/ta"
import { type Catalogue, detectLocale, translateIn } from "./index"

function keys(catalogue: Catalogue, prefix = ""): string[] {
  return Object.entries(catalogue).flatMap(([key, value]) => {
    const path = `${prefix}${key}`
    if (typeof value === "string" || typeof value.other === "string") return [path]
    return keys(value as Catalogue, `${path}.`)
  })
}

function placeholders(catalogue: Catalogue, key: string): string[] {
  const value = key.split(".").reduce<unknown>((node, part) => (node as Catalogue)[part], catalogue)
  const text = typeof value === "string" ? value : JSON.stringify(value)
  return [...new Set(text.match(/\{\w+\}/g) ?? [])].sort()
}

describe("catalogues", () => {
  const english = keys(en)

  it.each([
    ["si", si],
    ["ta", ta],
  ] as const)("%s gives every English key and nothing else", (_, catalogue) => {
    expect(keys(catalogue as Catalogue).sort()).toEqual([...english].sort())
  })

  it.each([
    ["si", si],
    ["ta", ta],
  ] as const)("%s keeps every placeholder of the English text", (_, catalogue) => {
    for (const key of english)
      expect(placeholders(catalogue as Catalogue, key), key).toEqual(placeholders(en, key))
  })

  it("has no empty message", () => {
    for (const catalogue of [en, si, ta] as Catalogue[])
      for (const key of keys(catalogue)) {
        const value = key
          .split(".")
          .reduce<unknown>((node, part) => (node as Catalogue)[part], catalogue)
        const texts = typeof value === "string" ? [value] : Object.values(value as object)
        for (const text of texts) expect(String(text).trim(), key).not.toBe("")
      }
  })
})

describe("translateIn", () => {
  it("reads a message in the language asked", () => {
    expect(translateIn("si", "common.language")).toBe("භාෂාව")
    expect(translateIn("en", "common.language")).toBe("Language")
  })
})

describe("detectLocale", () => {
  it("takes the first device language we have", () => {
    expect(detectLocale(["fr-FR", "ta-IN", "si-LK"])).toBe("ta")
    expect(detectLocale(["si"])).toBe("si")
  })
  it("falls back to English", () => {
    expect(detectLocale(["de-DE"])).toBe("en")
    expect(detectLocale([])).toBe("en")
  })
})
