/**
 * Which device and browser the page is open in, read from the user agent,
 * so the install guide can show the right steps first. Pure: everything
 * comes in as arguments, so the table of user agents can be tested.
 *
 * User agents lie in two well-known ways that matter here:
 *
 * - iPadOS 13 and later ask for desktop sites, so Safari on an iPad sends a
 *   Mac user agent. A Mac has no touch screen; an iPad reports several
 *   touch points.
 * - In-app browsers (Facebook, Instagram, TikTok, LINE and others) add their
 *   own token to an otherwise ordinary Safari or Chrome user agent. They
 *   cannot install web apps, so they are checked first.
 */

export type Os = "ios" | "android" | "macos" | "windows" | "linux" | "chromeos" | "other"

export type Browser = "safari" | "chrome" | "edge" | "firefox" | "samsung" | "opera" | "other"

/** Apps whose built-in browser cannot install a web app. */
export type InAppBrowser =
  | "facebook"
  | "messenger"
  | "instagram"
  | "tiktok"
  | "line"
  | "whatsapp"
  | "snapchat"
  | "wechat"
  | "twitter"
  | "linkedin"
  | "viber"
  | "google"
  | "webview"

export interface Platform {
  os: Os
  browser: Browser
  /** iPad, including iPadOS sending a Mac user agent. */
  tablet: boolean
  /** Set when the page is inside another app's browser. */
  inApp: InAppBrowser | null
  /** The page is already running as an installed app. */
  installed: boolean
}

export interface PlatformInput {
  userAgent: string
  /** `navigator.maxTouchPoints`: tells an iPad from a Mac. */
  maxTouchPoints?: number
  /** `display-mode: standalone` (or fullscreen, minimal-ui) matched, or `navigator.standalone`. */
  standalone?: boolean
}

/** In-app browser tokens, most specific first (Messenger also says FBAN). */
const inAppTokens: [RegExp, InAppBrowser][] = [
  [/FBAN\/Messenger|FB_IAB\/Orca|MessengerLite|FBAN\/MessengerForiOS/i, "messenger"],
  [/Instagram/i, "instagram"],
  [/FBAN|FBAV|FB_IAB|FBIOS|FB4A/i, "facebook"],
  [/musical_ly|BytedanceWebview|TikTok|trill_/i, "tiktok"],
  [/\bLine\/\d/i, "line"],
  [/WhatsApp/i, "whatsapp"],
  [/Snapchat/i, "snapchat"],
  [/MicroMessenger/i, "wechat"],
  [/\bTwitter/i, "twitter"],
  [/LinkedInApp/i, "linkedin"],
  [/Viber/i, "viber"],
  // The Google app on iPhone (GSA) opens links in its own view.
  [/\bGSA\/\d/i, "google"],
]

function detectInApp(ua: string, os: Os): InAppBrowser | null {
  for (const [pattern, app] of inAppTokens) if (pattern.test(ua)) return app
  // Android System WebView marks itself with "; wv)"; other apps embed it without naming themselves.
  if (os === "android" && /;\s*wv\)/.test(ua)) return "webview"
  return null
}

function detectOs(ua: string, maxTouchPoints: number): { os: Os; tablet: boolean } {
  if (/iPad/.test(ua)) return { os: "ios", tablet: true }
  if (/iPhone|iPod/.test(ua)) return { os: "ios", tablet: false }
  if (/Android/i.test(ua)) return { os: "android", tablet: !/Mobile/i.test(ua) }
  if (/CrOS/.test(ua)) return { os: "chromeos", tablet: false }
  if (/Macintosh|Mac OS X/.test(ua)) {
    // iPadOS asks for desktop sites with a Mac user agent; a Mac has no touch screen.
    return maxTouchPoints > 1 ? { os: "ios", tablet: true } : { os: "macos", tablet: false }
  }
  if (/Windows/.test(ua)) return { os: "windows", tablet: false }
  if (/Linux|X11/.test(ua)) return { os: "linux", tablet: false }
  return { os: "other", tablet: false }
}

function detectBrowser(ua: string, os: Os): Browser {
  if (/SamsungBrowser/i.test(ua)) return "samsung"
  if (/EdgiOS|EdgA|Edg\//.test(ua)) return "edge"
  if (/OPR\/|OPT\/|Opera|OPiOS/.test(ua)) return "opera"
  if (/FxiOS|Firefox\//.test(ua)) return "firefox"
  if (/CriOS|Chrome\/|Chromium\//.test(ua)) return "chrome"
  // Every iOS browser is WebKit; one with no token of its own is Safari.
  if (/Safari\//.test(ua) && (os === "ios" || os === "macos")) return "safari"
  if (os === "ios" && /AppleWebKit/.test(ua)) return "safari"
  return "other"
}

export function detectPlatform({
  userAgent,
  maxTouchPoints = 0,
  standalone = false,
}: PlatformInput): Platform {
  const { os, tablet } = detectOs(userAgent, maxTouchPoints)
  return {
    os,
    browser: detectBrowser(userAgent, os),
    tablet,
    inApp: detectInApp(userAgent, os),
    installed: standalone,
  }
}

/** The platform of this page, read from the browser. */
export function currentPlatform(): Platform {
  if (typeof navigator === "undefined") {
    return { os: "other", browser: "other", tablet: false, inApp: null, installed: false }
  }
  return detectPlatform({
    userAgent: navigator.userAgent,
    maxTouchPoints: navigator.maxTouchPoints ?? 0,
    standalone: isStandalone(),
  })
}

/** Whether the page runs as an installed app (any display mode but a browser tab). */
export function isStandalone(): boolean {
  if (typeof window === "undefined") return false
  const iosStandalone = (navigator as Navigator & { standalone?: boolean }).standalone === true
  const modes = ["standalone", "fullscreen", "minimal-ui", "window-controls-overlay"]
  return (
    iosStandalone ||
    modes.some((mode) => window.matchMedia?.(`(display-mode: ${mode})`).matches === true)
  )
}

/** Devices the guide has steps for, and the browsers it covers on each. */
export type Device = "iphone" | "android" | "computer"

export const guideBrowsers = {
  iphone: ["safari", "chrome", "edge", "firefox"],
  android: ["chrome", "samsung", "edge", "firefox"],
  computer: ["chrome", "edge", "safari", "firefox"],
} as const satisfies Record<Device, readonly Browser[]>

export type GuideBrowser<D extends Device = Device> = (typeof guideBrowsers)[D][number]

/** One set of steps: a device and a browser on it. */
export type GuideId = {
  [D in Device]: `${D}-${GuideBrowser<D>}`
}[Device]

export const guideIds: readonly GuideId[] = (Object.keys(guideBrowsers) as Device[]).flatMap(
  (device) => guideBrowsers[device].map((browser) => `${device}-${browser}` as GuideId),
)

/** What the guide opens on: the steps for this browser, or a special case. */
export type GuideStart =
  | { kind: "installed" }
  | { kind: "in-app"; app: InAppBrowser; os: Os }
  | { kind: "steps"; guide: GuideId; exact: boolean }

export function deviceOf(platform: Platform): Device {
  if (platform.os === "ios") return "iphone"
  if (platform.os === "android") return "android"
  return "computer"
}

/**
 * The steps to show first. `exact` is false when we fell back to the
 * nearest guide (for example Opera, or Chrome on Linux, which install like
 * Chrome) so the guide can say so.
 */
export function guideStart(platform: Platform): GuideStart {
  if (platform.installed) return { kind: "installed" }
  if (platform.inApp) return { kind: "in-app", app: platform.inApp, os: platform.os }
  const device = deviceOf(platform)
  const browsers = guideBrowsers[device] as readonly Browser[]
  if (browsers.includes(platform.browser)) {
    if (device === "computer" && platform.browser === "safari" && platform.os !== "macos") {
      return { kind: "steps", guide: "computer-chrome", exact: false }
    }
    return { kind: "steps", guide: `${device}-${platform.browser}` as GuideId, exact: true }
  }
  // Opera, Brave, Vivaldi and other Chromium browsers install the way Chrome does.
  return { kind: "steps", guide: `${device}-chrome` as GuideId, exact: false }
}

/**
 * Whether this browser can install the app at all, for the gentle prompt:
 * no point suggesting it in Firefox on a computer or inside another app.
 */
export function canInstall(platform: Platform): boolean {
  if (platform.installed || platform.inApp) return false
  if (deviceOf(platform) === "computer") {
    if (platform.browser === "firefox" || platform.browser === "other") return false
    if (platform.browser === "safari") return platform.os === "macos"
  }
  return true
}
