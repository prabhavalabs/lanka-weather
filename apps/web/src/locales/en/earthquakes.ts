/** Messages of the earthquakes feature. English is the source: it fixes the keys. */
export const earthquakes = {
  title: "Earthquakes",
  earthquake: "Earthquake",
  newTab: "(opens in a new tab)",
  /** "M6.2 earthquake" */
  hoverTitle: "M{magnitude} earthquake",
  /** "M6.2 120 km SW of Sumatra"; the place is USGS's. */
  listTitle: "M{magnitude} {place}",
  /** "1,605 km from Sri Lanka" */
  fromSriLanka: "{distance} from Sri Lanka",
  missing:
    "This event is not in the last 30 days of USGS's list. It may have been revised or deleted.",
  age: {
    day: "Last 24 hours",
    week: "Last 7 days",
    month: "Last 30 days",
  },
  summary: {
    none: "None of M4.5+ in {days} days",
    count: "{count} in {days} days",
    strongest: "strongest M{magnitude}",
  },
  tsunami:
    "USGS has set its tsunami flag on this event. The flag is an indicator that the event is in an oceanic region and large enough to look at; it is not a warning for Sri Lanka. For tsunami warnings, follow the Disaster Management Centre and the Department of Meteorology.",
  /** USGS's review status of an event. */
  status: {
    reviewed: "Reviewed",
    automatic: "Automatic",
    deleted: "Deleted",
  },
  legend: {
    size: "Circle size is magnitude",
    tsunami: "USGS tsunami flag, not a warning",
  },
  panel: {
    subtitle: "Earthquake · USGS",
    close: "Close earthquake details",
    footer: "Source: U.S. Geological Survey. Revised {revised}, fetched {fetched}.",
    magnitude: "Magnitude",
    time: "Time",
    depth: "Depth",
    fromCoast: "From Sri Lanka's coast",
    pager: "USGS PAGER alert",
    eventPage: "USGS event page",
    noWarnings:
      "Lanka Weather generates no tsunami warnings. Official warnings come from the DMC and the Department of Meteorology.",
  },
  details: {
    latest: "Latest",
    showOcean: "Show the Indian Ocean",
    coverage:
      "M4.5 and above from 15° S to 25° N and 60° to 105° E, and whatever smaller USGS lists near Sri Lanka. Lanka Weather generates no tsunami warnings.",
  },
}
