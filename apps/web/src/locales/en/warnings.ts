/** Messages of the warnings feature. English is the source: it fixes the keys. */
export const warnings = {
  newTab: "(opens in a new tab)",
  officialPdf: "Official PDF",
  officialReport: "Official report",
  dmcPage: "DMC page",
  withdrawn: "Withdrawn",
  /** A bulletin's badge: "Warning · Red". */
  badgeColour: "{severity} · {colour}",
  issuer: {
    nbroViaDmc: "NBRO, via DMC",
    /** "DMC · Weather" */
    withType: "{issuer} · {type}",
  },
  /** The language of a bulletin's edition, on its badge. */
  language: {
    en: "English",
    si: "Sinhala",
    ta: "Tamil",
  },
  /** The DMC's separate Sinhala and Tamil editions of a bulletin. */
  edition: {
    onlyEn: "Published in English only.",
    onlySi: "Published in Sinhala only.",
    onlyTa: "Published in Tamil only.",
    missingSi: "Sinhala edition not available.",
    missingTa: "Tamil edition not available.",
    englishText: "This is the English edition's text.",
    link: {
      en: "English edition (PDF)",
      si: "Sinhala edition (PDF)",
      ta: "Tamil edition (PDF)",
    },
  },
  area: {
    dsDistrict: "{ds}, {district} District",
    district: "{district} District",
    mentioned: "Mentioned in bulletin",
    mentionedIn: "Mentioned in bulletin: {names}",
    sriLanka: "Sri Lanka",
  },
  /** Why a district is outlined, in the hover chip. */
  highlight: {
    landslide: "Landslide warning: {level}",
    mentioned: "Mentioned in bulletin: {title}",
    more: { one: "and {count} more", other: "and {count} more" },
  },
  /** The landslide layer's one line in the layer list. */
  summary: {
    waiting: "Not collected yet",
    levelCount: "Level {level}: {count}",
    none: "None active as far as we know · last {date}",
    unknown: "Cannot say · feed out of date",
  },
  /** What can be said about the latest landslide report (warnings.Assess). */
  status: {
    waiting: "Landslide warnings have not been collected yet.",
    active: "Report of {date}, valid until {until}.",
    lapsed:
      "Report of {date}: its validity ended {until}. A newer report may not have reached us yet, so its levels stay shown.",
    unknown:
      "Last report {date}. We have not been able to check for newer reports since {checked}, so we cannot say whether a warning is active. Check with the DMC.",
    unknownRecently:
      "Last report {date}. We have not been able to check for newer reports recently, so we cannot say whether a warning is active. Check with the DMC.",
    none: "Last report {date}; none currently active as far as we know.",
    pending:
      "A newer report has been issued, and its levels have not reached us yet. Read the official report.",
    staleSince: "We have not been able to check for newer reports since {checked}.",
    staleRecently: "We have not been able to check for newer reports recently.",
    notNamed: "{ds} is not named in the landslide report of {date}. That does not mean it is safe.",
    notNamedHere:
      "This place is not named in the landslide report of {date}. That does not mean it is safe.",
    limit:
      "NBRO can lower or end a level before a report's period runs out. We learn of that only from its next report, hours later, so a level shown here may already have been withdrawn. Check with the DMC.",
  },
  landslide: {
    title: "Landslide early warning",
    layerTitle: "Landslide warnings",
    /** "Level 2 (Amber) · Alert" */
    levelAction: "{level} · {action}",
    levelShort: "Level {level}",
    /** "Report of 31 Oct 2026: Level 3 (Red) 0 · Level 2 (Amber) 1 · …" */
    reportCounts: "Report of {date}: {counts}",
    levelCount: "{level} {count}",
    /** A division's label on the map: "L2 Yatinuwara". */
    mapLabel: "L{level} {name}",
    mostSerious: "Most serious divisions",
    latestReport: "Latest official report (PDF)",
    validityNote:
      "Reports state a validity of about 24 hours and are reissued while a risk lasts. A place not named is not a place said to be safe.",
  },
  legend: {
    landslide: "Landslide warning in force",
  },
  banner: {
    /** "Level 3 (Red): 2 DS divisions" */
    landslideDetail: {
      one: "{level}: {count} DS division",
      other: "{level}: {count} DS divisions",
    },
    landslideDetailMore: "{detail}; {level}: {count}",
    all: "All warnings",
    allMore: { one: "All warnings ({count} more)", other: "All warnings ({count} more)" },
    dismiss: "Dismiss this notice",
  },
  /** The two tabs of the warnings panel. */
  tabs: { label: "Warnings and alerts", official: "Official warnings", alerts: "Your alerts" },
  button: {
    text: "Warnings",
    label: {
      one: "Official warnings, {count} in force",
      other: "Official warnings, {count} in force",
    },
    labelStale: {
      one: "Official warnings, {count} in force, out of date",
      other: "Official warnings, {count} in force, out of date",
    },
  },
  /** One DS division's landslide warning. */
  division: {
    title: "Landslide warning",
    notFound: "This DS division could not be found.",
    subtitle: "DS division, {district} District",
    close: "Close landslide warning",
    footer: "Source: {source}. Updated {age}.",
    noLevel: "No level in force as far as we know",
    notNamed: "{name} is not named in this report. That does not mean it is safe.",
    reportDate: "Report date",
    validUntilAssumed: "Valid until (assumed)",
    message: "Message",
    officialReport: "Official DMC report",
    history: "Recent reports naming {name}",
    noHistory: "No report we hold names this division.",
  },
  panel: {
    title: "Official warnings",
    subtitle: "NBRO, Department of Meteorology and DMC",
    close: "Close official warnings",
    filterLabel: "Show bulletins of type",
    all: "All",
    showAll: "Show all",
    loading: "Loading official warnings",
    error:
      "Official warnings could not be loaded. Check with the Disaster Management Centre, {site}.",
    noMatch: "No bulletins match in the last 45 days.",
    notCollected: "Bulletins have not been collected yet.",
    inForce: "In force",
    latest: "Latest bulletins",
    sourceNote:
      "Bulletins reach us through nuuuwan/lk_dmc, usually within hours of being issued. The PDF on dmc.gov.lk is the official version.",
    updated: "Updated {age}.",
    updatedRebuilt: "Updated {age}, index rebuilt {rebuilt}.",
    meteoNote:
      "Department of Meteorology warnings are shown as the DMC republishes them; its own site has not been reachable from our collector.",
  },
  bulletin: {
    validUntil: "valid until {time}",
    withdrawnBy: "Withdrawn by the notice of {time}, which names this basin.",
    withdrawalNotice: "Withdrawal notice",
    mayBeWithdrawn:
      "This may already have been withdrawn. A notice of {time} withdrew flood warnings, and we cannot tell whether it includes this one, so it stays shown.",
    readOrCheck: "{notice} or check {site}.",
    readNotice: "Read the notice",
    withdraws: "Withdraws flood warnings for: {basins}.",
    basinsUnread:
      "We could not read which basins this notice lists, so earlier warnings stay shown. Read the PDF.",
    readExcerpt: "Read an excerpt",
    excerptNote:
      "English sentences copied unchanged from the PDF. Tables and maps are left out, so read the PDF for the whole bulletin.",
  },
  /** The official-warnings layer in the layer list. */
  details: {
    inForce: { one: "{count} in force", other: "{count} in force" },
    noDistrict: "no district highlighted",
    highlights:
      "Highlights come only from landslide levels and from warnings or advisories whose clean English text names a district by exact word. A bulletin whose text cannot be read cleanly highlights nothing, so an unmarked district is not a clear one.",
    read: "Read official warnings",
  },
}
