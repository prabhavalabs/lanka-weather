/** Messages of the regions feature. English is the source: it fixes the keys. */
export const regions = {
  /** What the region colours show, in the layer's controls. */
  variable: {
    temperature: { label: "Temperature now", short: "Temp" },
    rain: { label: "Rain today", short: "Rain" },
    wind: { label: "Strongest wind now", short: "Wind" },
  },
  /** "by district", filled into the legend titles. */
  by: {
    province: "by province",
    district: "by district",
    ds: "by DS division",
    gn: "by GN division",
  },
  legend: {
    temperature: "Temperature {by}",
    rain: "Rain today {by}",
    wind: "Max wind {by}",
  },
  /** A region's weather as a whole, in the place panel. */
  aggregate: {
    title: "{level} as a whole",
    meanNow: "Mean now",
    today: "Today",
    maxWind: "Max wind",
    rainNow: "Rain now",
    rainToday: "Rain today",
    maxGusts: "Max gusts",
    cells: {
      one: "Averaged over {count} grid cell, weighted by area.",
      other: "Averaged over {count} grid cells, weighted by area.",
    },
    point: "Read at a point inside the region.",
  },
}
