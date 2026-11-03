/**
 * The install guide (features/install). Short sentences for people who
 * have never installed a web app: name the button, then what to tap.
 * Button names in quotes are the browser's own words, in English, because
 * that is what most phones in Sri Lanka show.
 */
export const install = {
  open: "Install app",
  openHint: "Put Lanka Weather on your phone or computer",
  title: "Install Lanka Weather",
  description: "Add it to your phone or computer and open it like any other app.",
  alertsReason:
    "On iPhone and iPad, alerts work only after you add Lanka Weather to your Home Screen.",
  installNow: "Install",
  installNowHint: "Your browser can install it in one tap.",
  orSteps: "Or follow these steps:",
  stepsFor: "{device} · {browser}",
  nearest: "We are not sure which browser this is. These steps work in most browsers like it.",
  otherDevice: "Using a different phone or browser?",
  device: "Device",
  browser: "Browser",
  devices: {
    iphone: "iPhone or iPad",
    android: "Android",
    computer: "Computer",
  },
  browsers: {
    safari: "Safari",
    chrome: "Chrome",
    edge: "Edge",
    firefox: "Firefox",
    samsung: "Samsung Internet",
  },
  step: "Step {number}",
  steps: {
    "iphone-safari": {
      s1: "Tap the Share button: a square with an arrow pointing up. If you do not see it, tap ⋯ first.",
      s2: "Scroll down and tap “Add to Home Screen”.",
      s3: "Tap “Add” in the top corner.",
      s4: "Open Lanka Weather from the new icon on your Home Screen. Turn on alerts there.",
    },
    "iphone-chrome": {
      s1: "Tap the Share button in the address bar. Or tap ⋯, then “Share”.",
      s2: "Tap “Add to Home Screen”. Scroll down if you do not see it.",
      s3: "Tap “Add” in the top corner.",
      s4: "Open Lanka Weather from the new icon on your Home Screen. Turn on alerts there.",
    },
    "iphone-edge": {
      s1: "Tap ⋯ at the bottom of the screen, then “Share”.",
      s2: "Tap “Add to Home Screen”. Scroll down if you do not see it.",
      s3: "Tap “Add” in the top corner.",
      s4: "Open Lanka Weather from the new icon on your Home Screen. Turn on alerts there.",
    },
    "iphone-firefox": {
      s1: "Tap the menu button ☰, then “Share”.",
      s2: "Tap “Add to Home Screen”. Scroll down if you do not see it.",
      s3: "Tap “Add” in the top corner.",
      s4: "Open Lanka Weather from the new icon on your Home Screen. Turn on alerts there.",
    },
    "android-chrome": {
      s1: "Tap ⋮ at the top right.",
      s2: "Tap “Add to Home screen” or “Install app”.",
      s3: "Tap “Install”.",
      s4: "Open Lanka Weather from your Home screen or app list.",
    },
    "android-samsung": {
      s1: "Tap the menu button ☰ at the bottom right.",
      s2: "Tap “Add page to”, then “Home screen”.",
      s3: "Tap “Add”. If you see a download icon in the address bar, you can tap it and then “Install” instead.",
      s4: "Open Lanka Weather from your Home screen or app list.",
    },
    "android-edge": {
      s1: "Tap ⋯ at the bottom of the screen.",
      s2: "Tap “Add to phone” or “Install app”.",
      s3: "Tap “Install”.",
      s4: "Open Lanka Weather from your Home screen or app list.",
    },
    "android-firefox": {
      s1: "Tap ⋮ next to the address bar.",
      s2: "Tap “Add app to Home screen”. If you do not see it, look under “More”.",
      s3: "Tap “Add”.",
      s4: "Open Lanka Weather from your Home screen or app list.",
    },
    "computer-chrome": {
      s1: "Click the install icon at the right end of the address bar: a screen with a down arrow.",
      s2: "No icon? Click ⋮, then “Cast, save, and share”, then “Install Lanka Weather”.",
      s3: "Click “Install”. Lanka Weather opens in its own window.",
    },
    "computer-edge": {
      s1: "Click the “App available” icon in the address bar.",
      s2: "No icon? Click ⋯, then “Apps”, then “Install Lanka Weather”.",
      s3: "Click “Install”. Lanka Weather opens in its own window.",
    },
    "computer-safari": {
      s1: "Click “File” in the menu bar at the top of the screen.",
      s2: "Click “Add to Dock”.",
      s3: "Click “Add”. Lanka Weather is now in your Dock.",
    },
    "computer-firefox": {
      s1: "Firefox on a computer cannot install web apps yet.",
      s2: "Open this page in Chrome or Edge, or in Safari on a Mac, and install it there.",
      s3: "Or keep it as a bookmark: press Ctrl+D (⌘D on a Mac).",
    },
  },
  notes: {
    iosVersion: "Alerts need iOS or iPadOS 16.4 or later.",
    macVersion: "Needs macOS Sonoma (14) or later.",
  },
  inApp: {
    title: "Open this page in your browser",
    body: "{app} opened this page in its own browser, which cannot install apps.",
    bodyUnknown: "This app opened this page in its own browser, which cannot install apps.",
    ios: "Tap ⋯ or the Share button, then “Open in Safari”. Or copy the link and paste it into Safari.",
    android:
      "Tap ⋮, then “Open in Chrome” or “Open in browser”. Or copy the link and paste it into Chrome.",
    copy: "Copy link",
    copied: "Link copied. Paste it into your browser.",
    copyFailed: "Could not copy. Press and hold the address to copy it.",
    openChrome: "Open in Chrome",
  },
  installed: {
    title: "Lanka Weather is installed",
    body: "You are using the installed app. It opens full screen and keeps the last data for when you are offline.",
    alerts: "Turn on alerts to hear about warnings for your area.",
    alertSettings: "Alert settings",
  },
  why: {
    title: "Why install?",
    alerts: "Alerts about warnings near you, on iPhone too",
    fullScreen: "Opens full screen, like an app",
    offline: "Works offline with the last saved data",
    small: "Small and free, with no app store",
  },
  prompt: {
    title: "Install Lanka Weather?",
    body: "Get alerts and use the map offline.",
    show: "Show me how",
    later: "Not now",
  },
  thanks: {
    title: "Thanks for installing Lanka Weather",
    body: "Open it from your Home Screen or app list.",
  },
  close: "Close",
}
