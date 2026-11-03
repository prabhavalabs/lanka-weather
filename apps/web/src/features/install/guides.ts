import type { MessageKey } from "@/lib/i18n"
import type { StepPicture } from "./Pictogram"
import type { Device, GuideId } from "./platform"

export interface GuideStep {
  text: MessageKey
  picture: StepPicture
}

export interface Guide {
  steps: GuideStep[]
  /** A requirement shown under the steps. */
  note?: MessageKey
}

const openFromHome: StepPicture = { glyph: "app" }

/** The steps for every device and browser, in the browser's own words (locales/en/install.ts). */
export const guides: Record<GuideId, Guide> = {
  "iphone-safari": {
    steps: [
      { text: "install.steps.iphone-safari.s1", picture: { glyph: "share" } },
      { text: "install.steps.iphone-safari.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.iphone-safari.s3", picture: { label: "Add" } },
      { text: "install.steps.iphone-safari.s4", picture: openFromHome },
    ],
    note: "install.notes.iosVersion",
  },
  "iphone-chrome": {
    steps: [
      { text: "install.steps.iphone-chrome.s1", picture: { glyph: "share" } },
      { text: "install.steps.iphone-chrome.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.iphone-chrome.s3", picture: { label: "Add" } },
      { text: "install.steps.iphone-chrome.s4", picture: openFromHome },
    ],
    note: "install.notes.iosVersion",
  },
  "iphone-edge": {
    steps: [
      { text: "install.steps.iphone-edge.s1", picture: { glyph: "ellipsis" } },
      { text: "install.steps.iphone-edge.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.iphone-edge.s3", picture: { label: "Add" } },
      { text: "install.steps.iphone-edge.s4", picture: openFromHome },
    ],
    note: "install.notes.iosVersion",
  },
  "iphone-firefox": {
    steps: [
      { text: "install.steps.iphone-firefox.s1", picture: { glyph: "menu" } },
      { text: "install.steps.iphone-firefox.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.iphone-firefox.s3", picture: { label: "Add" } },
      { text: "install.steps.iphone-firefox.s4", picture: openFromHome },
    ],
    note: "install.notes.iosVersion",
  },
  "android-chrome": {
    steps: [
      { text: "install.steps.android-chrome.s1", picture: { glyph: "kebab" } },
      { text: "install.steps.android-chrome.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.android-chrome.s3", picture: { label: "Install" } },
      { text: "install.steps.android-chrome.s4", picture: openFromHome },
    ],
  },
  "android-samsung": {
    steps: [
      { text: "install.steps.android-samsung.s1", picture: { glyph: "menu" } },
      { text: "install.steps.android-samsung.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.android-samsung.s3", picture: { label: "Add" } },
      { text: "install.steps.android-samsung.s4", picture: openFromHome },
    ],
  },
  "android-edge": {
    steps: [
      { text: "install.steps.android-edge.s1", picture: { glyph: "ellipsis" } },
      { text: "install.steps.android-edge.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.android-edge.s3", picture: { label: "Install" } },
      { text: "install.steps.android-edge.s4", picture: openFromHome },
    ],
  },
  "android-firefox": {
    steps: [
      { text: "install.steps.android-firefox.s1", picture: { glyph: "kebab" } },
      { text: "install.steps.android-firefox.s2", picture: { glyph: "add-square" } },
      { text: "install.steps.android-firefox.s3", picture: { label: "Add" } },
      { text: "install.steps.android-firefox.s4", picture: openFromHome },
    ],
  },
  "computer-chrome": {
    steps: [
      { text: "install.steps.computer-chrome.s1", picture: { glyph: "install-desktop" } },
      { text: "install.steps.computer-chrome.s2", picture: { glyph: "kebab" } },
      { text: "install.steps.computer-chrome.s3", picture: { label: "Install" } },
    ],
  },
  "computer-edge": {
    steps: [
      { text: "install.steps.computer-edge.s1", picture: { glyph: "app-available" } },
      { text: "install.steps.computer-edge.s2", picture: { glyph: "ellipsis" } },
      { text: "install.steps.computer-edge.s3", picture: { label: "Install" } },
    ],
  },
  "computer-safari": {
    steps: [
      { text: "install.steps.computer-safari.s1", picture: { label: "File" } },
      { text: "install.steps.computer-safari.s2", picture: { glyph: "dock" } },
      { text: "install.steps.computer-safari.s3", picture: { label: "Add" } },
    ],
    note: "install.notes.macVersion",
  },
  "computer-firefox": {
    steps: [
      { text: "install.steps.computer-firefox.s1", picture: { glyph: "browser" } },
      { text: "install.steps.computer-firefox.s2", picture: { glyph: "install-desktop" } },
      { text: "install.steps.computer-firefox.s3", picture: { glyph: "bookmark" } },
    ],
  },
}

export function splitGuide(id: GuideId): { device: Device; browser: string } {
  const [device, browser] = id.split("-") as [Device, string]
  return { device, browser }
}
