/**
 * The interface in English, Sinhala and Tamil.
 *
 * Messages live in `src/locales/<locale>/<namespace>.ts`; English is the
 * source and fixes the keys, and Sinhala and Tamil must give every key (a
 * test checks). A missing translation falls back to English rather than
 * showing a key.
 *
 * `translate` reads the current language, so plain helpers can call it while
 * a component renders. Components call `useT()` (or `useLocale()`), which
 * subscribes them, so they re-render when the language changes.
 */
import type { ExpressionSpecification } from "maplibre-gl"
import { useCallback, useSyncExternalStore } from "react"
import { en } from "@/locales/en"
import { si } from "@/locales/si"
import { ta } from "@/locales/ta"
import { type Locale, isLocale, localeStorageKey, locales } from "./locales"

export { isLocale, localeStorageKey, locales, localeOrder, type Locale } from "./locales"

/** A message: plain text, or text per plural category for a `count`. */
export type Message = string | { one?: string; other: string; zero?: string }

/** A catalogue: namespaces of nested messages. */
export interface Catalogue {
  [key: string]: Message | Catalogue
}

type Leaf<T> = T extends string ? true : T extends { other: string } ? true : false

/** Dotted paths to every message, e.g. "install.ios.step1". */
export type MessageKey<T = typeof en, Prefix extends string = ""> = {
  [K in keyof T & string]: Leaf<T[K]> extends true
    ? `${Prefix}${K}`
    : MessageKey<T[K], `${Prefix}${K}.`>
}[keyof T & string]

/** Values for `{name}` placeholders; `count` also picks the plural form. */
export type MessageValues = Record<string, string | number>

/** The shape a translation must follow: every key of English, as text. */
export type Translation<T> = {
  [K in keyof T]: T[K] extends string
    ? string
    : T[K] extends { other: string }
      ? Message
      : Translation<T[K]>
}

const catalogues: Record<Locale, Catalogue> = { en, si, ta }

const listeners = new Set<() => void>()

function readStoredLocale(): Locale | null {
  try {
    const stored = localStorage.getItem(localeStorageKey)
    return isLocale(stored) ? stored : null
  } catch {
    return null
  }
}

/** The device's language when it is one we have, else English. */
export function detectLocale(languages: readonly string[] = navigator.languages ?? []): Locale {
  for (const language of languages) {
    const base = language.toLowerCase().split("-")[0]
    if (isLocale(base)) return base
  }
  return "en"
}

let current: Locale = typeof window === "undefined" ? "en" : (readStoredLocale() ?? detectLocale())

function apply() {
  if (typeof document === "undefined") return
  document.documentElement.lang = locales[current].tag
}
apply()

export function getLocale(): Locale {
  return current
}

/** Changes the language and remembers it on this device. */
export function setLocale(next: Locale) {
  if (next === current) return
  current = next
  try {
    localStorage.setItem(localeStorageKey, next)
  } catch {
    // Storage may be disabled; the choice then lasts for the session.
  }
  apply()
  for (const listener of listeners) listener()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

/** The current language; the component re-renders when it changes. */
export function useLocale(): Locale {
  return useSyncExternalStore(subscribe, getLocale, () => "en" as Locale)
}

function lookup(catalogue: Catalogue, key: string): Message | undefined {
  let node: Message | Catalogue | undefined = catalogue
  for (const part of key.split(".")) {
    if (node === undefined || typeof node === "string") return undefined
    node = (node as Catalogue)[part]
  }
  if (typeof node === "string") return node
  if (node && typeof node === "object" && typeof node.other === "string") return node as Message
  return undefined
}

const pluralRules = new Map<Locale, Intl.PluralRules>()

function pick(message: Message, locale: Locale, count: number | undefined): string {
  if (typeof message === "string") return message
  if (count === undefined) return message.other
  if (count === 0 && message.zero !== undefined) return message.zero
  let rules = pluralRules.get(locale)
  if (!rules) {
    rules = new Intl.PluralRules(locales[locale].tag)
    pluralRules.set(locale, rules)
  }
  const category = rules.select(count)
  return (category === "one" ? message.one : undefined) ?? message.other
}

/** Fills `{name}` placeholders; an unknown one is left as written. */
function fill(text: string, values: MessageValues | undefined): string {
  if (!values) return text
  return text.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in values ? String(values[name]) : match,
  )
}

/** Whether a message exists, e.g. for words keyed by an id the API may add to. */
export function hasMessage(key: string, locale: Locale = "en"): boolean {
  return lookup(catalogues[locale], key) !== undefined
}

/** A message in a given language, falling back to English. */
export function translateIn(locale: Locale, key: MessageKey, values?: MessageValues): string {
  const message = lookup(catalogues[locale], key) ?? lookup(catalogues.en, key)
  if (message === undefined) return key
  const count = typeof values?.count === "number" ? values.count : undefined
  return fill(pick(message, locale, count), values)
}

/** A message in the current language. Call it while rendering a subscribed component. */
export function translate(key: MessageKey, values?: MessageValues): string {
  return translateIn(current, key, values)
}

export type Translate = typeof translate

/** `t` for a component: subscribes it to the language. */
export function useT(): Translate {
  const locale = useLocale()
  return useCallback(
    (key: MessageKey, values?: MessageValues) => translateIn(locale, key, values),
    [locale],
  )
}

/** The `Intl` tag for dates and numbers in the current language, e.g. "si-LK". */
export function intlTag(locale: Locale = current): string {
  return locales[locale].tag
}

/**
 * A name as the API gives it: English, with Sinhala and Tamil where the
 * data has them. Proper names are never translated here, only chosen.
 */
export interface LocalNames {
  name: string
  nameSi?: string | null
  nameTa?: string | null
}

/** The API's `names` object: `{ en, si, ta }`. */
export interface NameSet {
  en: string
  si?: string | null
  ta?: string | null
}

function usable(name: string | null | undefined): name is string {
  return typeof name === "string" && name.trim() !== ""
}

/** The name in the language, else the English name. */
export function localName(names: LocalNames, locale: Locale = current): string {
  const local = locale === "si" ? names.nameSi : locale === "ta" ? names.nameTa : undefined
  return usable(local) ? local : names.name
}

/** The name in the language from a `{ en, si, ta }` set, else the English one. */
export function nameIn(names: NameSet, locale: Locale = current): string {
  return localName({ name: names.en, nameSi: names.si, nameTa: names.ta }, locale)
}

/**
 * A MapLibre expression for a feature's name in the language, falling back
 * to the English property. Layers rebuild it when the language changes.
 */
export function localNameExpression(
  locale: Locale = current,
  fields: { name: string; si?: string; ta?: string } = {
    name: "name",
    si: "nameSi",
    ta: "nameTa",
  },
): ExpressionSpecification {
  const local = locale === "si" ? fields.si : locale === "ta" ? fields.ta : undefined
  if (!local) return ["to-string", ["coalesce", ["get", fields.name], ""]]
  return ["to-string", ["coalesce", ["get", local], ["get", fields.name], ""]]
}

/** Folds a name for matching: case, spacing and joiners do not count. */
export function foldName(name: string): string {
  return name.normalize("NFC").replace(/[‌‍]/g, "").toLowerCase().replace(/\s+/g, " ").trim()
}

/** Whether any of a feature's names contains the query, in any script. */
export function namesMatch(names: Array<string | null | undefined>, query: string): boolean {
  const folded = foldName(query)
  if (folded === "") return false
  return names.some((name) => usable(name) && foldName(name).includes(folded))
}
