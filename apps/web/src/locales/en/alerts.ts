/** Alert settings: Web Push notices on this device (ADR 0018). */
export const alerts = {
  button: "Alerts",
  buttonOn: "Alerts on",
  buttonLabel: "Alert settings",
  title: "Alerts on this device",
  description:
    "Get warnings as notifications, even when Lanka Weather is closed. No account needed.",

  status: {
    checking: "Checking whether this device can receive alerts…",
    unsupported:
      "This browser cannot receive alerts. Try a recent Chrome, Edge, Firefox or Samsung Internet, or Safari on iPhone with the app on the Home Screen.",
    iosInstallTitle: "Add Lanka Weather to your Home Screen first",
    iosInstall:
      "On iPhone and iPad, alerts work only from the app on the Home Screen (iOS 16.4 or later). Add it, open it from the Home Screen, then turn alerts on here.",
    iosInstallAction: "How to add it",
    disabled: "Alerts are not switched on for this service yet. Check back soon.",
    deniedTitle: "Notifications are blocked for this site",
    denied: "Allow notifications for this site, then come back here.",
    unavailable:
      "Alerts could not start on this device: the app's background worker is not ready. Reload the page and try again.",
    off: "Alerts are off on this device.",
    on: "Alerts are on. You will be notified about what you choose below.",
    onSince: "On since {date}",
  },

  denied: {
    chrome:
      "Chrome or Edge: tap the icon left of the address, choose Permissions or Site settings, and set Notifications to Allow.",
    firefox:
      "Firefox: tap the icon left of the address, then clear the blocked Notifications permission.",
    safari:
      "Safari on a Mac: Safari menu, Settings, Websites, Notifications, and allow this site. On iPhone: Settings, Notifications, Lanka Weather.",
    android:
      "Android: long-press the app or open the browser's Site settings, Notifications, and allow this site.",
    generic: "Open your browser's site settings and allow notifications for this site.",
  },

  retry: "Try again",
  turnOn: "Turn on alerts",
  turnOff: "Turn off alerts",
  turningOn: "Turning on…",
  sendTest: "Send a test alert",
  testSent: "Test alert sent. It should arrive in a few seconds.",
  testTooSoon: "A test was sent a moment ago. Wait a minute before sending another.",
  dismissed: "Notifications were not allowed. Tap Turn on alerts to ask again.",
  failed: "That did not work. Check your connection and try again.",
  gone: "This device's alert subscription expired. Turn alerts on again.",
  saving: "Saving…",
  saved: "Saved",
  saveFailed: "Could not save your choices. They will be saved when you are back online.",

  areas: {
    title: "Where",
    country: "Whole country",
    chosen: "Choose areas",
    districts: "Districts",
    districtsCount: { one: "{count} district", other: "{count} districts" },
    districtsLoading: "Loading districts…",
    districtsFailed: "Could not load the districts.",
    nearMe: "Near me",
    nearMeHint: "Uses your location once, rounded to about 1 km. It is not tracked.",
    radius: "Within",
    km: "{km} km",
    locating: "Finding your location…",
    locationDenied: "Location was not allowed. Choose districts instead, or allow location.",
    locationOutside: "Your location is outside Sri Lanka. Choose districts instead.",
    locationFailed: "Could not find your location.",
    savedPoint: "Location saved near {lat}, {lon}",
    needOne: "Choose at least one district or turn on Near me.",
  },

  topics: {
    title: "What",
    needOne: "Choose at least one kind of alert.",
    official: "Official warnings",
    officialHint: "DMC and Department of Meteorology warnings and advisories",
    landslide: "Landslide Level 2 and 3",
    landslideHint: "NBRO: be alert, and evacuate",
    landslide_level1: "Landslide Level 1",
    landslide_level1Hint: "NBRO: watch. Off by default; sent whatever the level below",
    river: "River flood levels",
    riverHint: "Gauges at alert, minor flood and major flood level",
    heavy_rain: "Heavy rain measured",
    heavy_rainHint: "Bursts at the live rain gauges that can flood towns. Not official warnings",
    flood_forecast: "Flood forecasts",
    flood_forecastHint: "Google Flood Hub estimates, not official warnings",
    cyclone: "Cyclones",
    cycloneHint: "A cyclone within 1,000 km of Sri Lanka, or forecast to come within it",
    earthquake: "Large earthquakes",
    earthquakeHint: "Nearby or large ones. Never a tsunami warning",
    reservoir: "Reservoir spills",
    reservoirHint: "From waterlevels.lk, not official. Off by default; for your areas",
  },

  severity: {
    title: "How serious",
    hint: "Only send alerts at this level or above.",
    watch: "Watch",
    warning: "Warning",
    danger: "Danger",
    watchHint: "Everything, including alert levels and Level 1",
    warningHint: "Minor flood, Level 2, official warnings and above",
    dangerHint: "Only major flood, Level 3, red notices and the like",
  },

  note: "Official DMC, NBRO, Irrigation Department and Department of Meteorology notices are authoritative. Alerts can arrive late or not at all: never rely on them alone. In an emergency call 117.",
  privacy:
    "We keep this device's push address, your choices and language, and nothing else: no account, no IP address.",
}
