import { describe, expect, it } from "vitest"
import {
  canInstall,
  detectPlatform,
  guideIds,
  guideStart,
  type GuideStart,
  type PlatformInput,
} from "./platform"
import { guides } from "./guides"

const ua = {
  iphoneSafari:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1",
  iphoneChrome:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/138.0.7204.119 Mobile/15E148 Safari/604.1",
  iphoneEdge:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 EdgiOS/138.3351.83 Mobile/15E148 Safari/605.1.15",
  iphoneFirefox:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/140.0 Mobile/15E148 Safari/605.1.15",
  ipadOld:
    "Mozilla/5.0 (iPad; CPU OS 12_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.1.2 Mobile/15E148 Safari/604.1",
  // iPadOS 13+ asks for desktop sites: the same string as Safari on a Mac.
  macSafari:
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1.15",
  androidChrome:
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Mobile Safari/537.36",
  androidTabletChrome:
    "Mozilla/5.0 (Linux; Android 14; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
  samsung:
    "Mozilla/5.0 (Linux; Android 14; SAMSUNG SM-S921B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/28.0 Chrome/130.0.0.0 Mobile Safari/537.36",
  androidEdge:
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Mobile Safari/537.36 EdgA/138.0.3351.65",
  androidFirefox: "Mozilla/5.0 (Android 14; Mobile; rv:140.0) Gecko/140.0 Firefox/140.0",
  androidOpera:
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Mobile Safari/537.36 OPR/90.0.0.0",
  windowsChrome:
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
  windowsEdge:
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.3351.83",
  windowsFirefox:
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0",
  macChrome:
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
  linuxChrome:
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
  chromebook:
    "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
  // In-app browsers.
  facebookIos:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 [FBAN/FBIOS;FBAV/520.0.0.38.101;FBBV/752116583;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/18.5;FBSS/3;FBID/phone;FBLC/en_GB;FBOP/5]",
  facebookAndroid:
    "Mozilla/5.0 (Linux; Android 14; SM-A546E Build/UP1A.231005.007; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/138.0.7204.63 Mobile Safari/537.36 [FB_IAB/FB4A;FBAV/520.0.0.47.109;]",
  messengerAndroid:
    "Mozilla/5.0 (Linux; Android 14; Pixel 8 Build/AP2A.240905.003; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/138.0.7204.63 Mobile Safari/537.36 [FB_IAB/Orca-Android;FBAV/480.0.0.38.109;]",
  messengerIos:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 [FBAN/MessengerForiOS;FBAV/480.0.0.38.109;FBBV/673416063;FBDV/iPhone14,5;FBMD/iPhone;FBSN/iOS]",
  instagram:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 389.0.0.29.81 (iPhone15,3; iOS 18_5; en_GB; en-GB; scale=3.00; 1290x2796; 772150291)",
  tiktok:
    "Mozilla/5.0 (Linux; Android 13; SM-A135F Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/137.0.7151.115 Mobile Safari/537.36 trill_400305 JsSdk/1.0 NetType/WIFI Channel/googleplay AppName/musical_ly app_version/40.3.5 ByteLocale/en BytedanceWebview/d8a21c6",
  line: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Safari Line/15.10.0",
  whatsapp:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 WhatsApp/25.20.79",
  androidWebView:
    "Mozilla/5.0 (Linux; Android 14; Pixel 8 Build/AP2A.240905.003; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/138.0.7204.63 Mobile Safari/537.36",
  googleApp:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/376.0.779315418 Mobile/15E148 Safari/604.1",
}

function start(input: PlatformInput): GuideStart {
  return guideStart(detectPlatform(input))
}

describe("detectPlatform", () => {
  it.each([
    ["iPhone Safari", ua.iphoneSafari, "ios", "safari", false],
    ["iPhone Chrome", ua.iphoneChrome, "ios", "chrome", false],
    ["iPhone Edge", ua.iphoneEdge, "ios", "edge", false],
    ["iPhone Firefox", ua.iphoneFirefox, "ios", "firefox", false],
    ["old iPad Safari", ua.ipadOld, "ios", "safari", true],
    ["Android Chrome", ua.androidChrome, "android", "chrome", false],
    ["Android tablet Chrome", ua.androidTabletChrome, "android", "chrome", true],
    ["Samsung Internet", ua.samsung, "android", "samsung", false],
    ["Android Edge", ua.androidEdge, "android", "edge", false],
    ["Android Firefox", ua.androidFirefox, "android", "firefox", false],
    ["Android Opera", ua.androidOpera, "android", "opera", false],
    ["Windows Chrome", ua.windowsChrome, "windows", "chrome", false],
    ["Windows Edge", ua.windowsEdge, "windows", "edge", false],
    ["Windows Firefox", ua.windowsFirefox, "windows", "firefox", false],
    ["Mac Chrome", ua.macChrome, "macos", "chrome", false],
    ["Mac Safari", ua.macSafari, "macos", "safari", false],
    ["Linux Chrome", ua.linuxChrome, "linux", "chrome", false],
    ["Chromebook", ua.chromebook, "chromeos", "chrome", false],
  ] as const)("%s", (_, userAgent, os, browser, tablet) => {
    const platform = detectPlatform({ userAgent })
    expect(platform).toMatchObject({ os, browser, tablet, inApp: null, installed: false })
  })

  it("tells iPadOS asking for the desktop site from a Mac by its touch points", () => {
    expect(detectPlatform({ userAgent: ua.macSafari, maxTouchPoints: 5 })).toMatchObject({
      os: "ios",
      browser: "safari",
      tablet: true,
    })
    expect(detectPlatform({ userAgent: ua.macSafari, maxTouchPoints: 0 }).os).toBe("macos")
    // Chrome on an iPad in desktop mode keeps its CriOS token only on iPhone; on iPad it sends Mac Safari's string too.
    expect(start({ userAgent: ua.macSafari, maxTouchPoints: 5 })).toEqual({
      kind: "steps",
      guide: "iphone-safari",
      exact: true,
    })
  })

  it.each([
    ["Facebook on iPhone", ua.facebookIos, "facebook", "ios"],
    ["Facebook on Android", ua.facebookAndroid, "facebook", "android"],
    ["Messenger on Android", ua.messengerAndroid, "messenger", "android"],
    ["Messenger on iPhone", ua.messengerIos, "messenger", "ios"],
    ["Instagram", ua.instagram, "instagram", "ios"],
    ["TikTok", ua.tiktok, "tiktok", "android"],
    ["LINE", ua.line, "line", "ios"],
    ["WhatsApp's viewer", ua.whatsapp, "whatsapp", "ios"],
    ["an Android WebView", ua.androidWebView, "webview", "android"],
    ["the Google app", ua.googleApp, "google", "ios"],
  ] as const)("finds %s as an in-app browser", (_, userAgent, app, os) => {
    const platform = detectPlatform({ userAgent })
    expect(platform.inApp).toBe(app)
    expect(platform.os).toBe(os)
    expect(guideStart(platform)).toEqual({ kind: "in-app", app, os })
    expect(canInstall(platform)).toBe(false)
  })

  it("does not take ordinary browsers for in-app ones", () => {
    for (const userAgent of [ua.iphoneSafari, ua.iphoneChrome, ua.androidChrome, ua.samsung])
      expect(detectPlatform({ userAgent }).inApp).toBeNull()
  })
})

describe("guideStart", () => {
  it.each([
    [ua.iphoneSafari, "iphone-safari", true],
    [ua.iphoneChrome, "iphone-chrome", true],
    [ua.iphoneEdge, "iphone-edge", true],
    [ua.iphoneFirefox, "iphone-firefox", true],
    [ua.androidChrome, "android-chrome", true],
    [ua.samsung, "android-samsung", true],
    [ua.androidEdge, "android-edge", true],
    [ua.androidFirefox, "android-firefox", true],
    [ua.androidOpera, "android-chrome", false],
    [ua.windowsChrome, "computer-chrome", true],
    [ua.windowsEdge, "computer-edge", true],
    [ua.windowsFirefox, "computer-firefox", true],
    [ua.macSafari, "computer-safari", true],
    [ua.linuxChrome, "computer-chrome", true],
  ] as const)("picks the steps for %s", (userAgent, guide, exact) => {
    expect(start({ userAgent })).toEqual({ kind: "steps", guide, exact })
  })

  it("says so when the app is already installed", () => {
    expect(start({ userAgent: ua.iphoneSafari, standalone: true })).toEqual({ kind: "installed" })
    expect(canInstall(detectPlatform({ userAgent: ua.androidChrome, standalone: true }))).toBe(
      false,
    )
  })

  it("has steps for every guide", () => {
    for (const id of guideIds) expect(guides[id].steps.length, id).toBeGreaterThan(1)
  })
})

describe("canInstall", () => {
  it.each([
    [ua.iphoneSafari, true],
    [ua.iphoneChrome, true],
    [ua.androidChrome, true],
    [ua.samsung, true],
    [ua.androidFirefox, true],
    [ua.windowsChrome, true],
    [ua.windowsEdge, true],
    [ua.macSafari, true],
    [ua.windowsFirefox, false],
  ] as const)("%s → %s", (userAgent, expected) => {
    expect(canInstall(detectPlatform({ userAgent }))).toBe(expected)
  })
})
