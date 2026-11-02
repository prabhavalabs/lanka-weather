import { describe, expect, it } from "vitest"
import { RAIN_GAUGE_POINTS, RAIN_MET_POINTS } from "../rainfall/ObservedRainLayer"
import { MET_STATIONS } from "../stations/MetStationsLayer"
import { observedChip, observedTarget } from "./points"

const hit = (layer: string, properties: Record<string, unknown>) => ({
  layer: { id: layer },
  properties,
})

describe("observedTarget", () => {
  it("opens a Met station from either layer", () => {
    expect(observedTarget(hit(MET_STATIONS, { id: "galle" }))).toEqual({ kind: "met", id: "galle" })
    expect(observedTarget(hit(RAIN_MET_POINTS, { stationId: "galle" }))).toEqual({
      kind: "met",
      id: "galle",
    })
  })

  it("opens a gauge's river station", () => {
    expect(observedTarget(hit(RAIN_GAUGE_POINTS, { stationId: "ampara" }))).toEqual({
      kind: "station",
      id: "ampara",
    })
  })

  it("leaves other layers to the caller", () => {
    expect(observedTarget(hit("river-levels-circles", { id: "x" }))).toBeNull()
  })
})

describe("observedChip", () => {
  it("says a temperature was observed, and when", () => {
    const chip = observedChip(
      hit(MET_STATIONS, {
        name: "Colombo",
        shown: 28.1,
        reading: "3h",
        observedAt: null,
        stale: true,
      }),
    )
    expect(chip?.title).toBe("Colombo")
    expect(chip?.lines[0]).toBe("Observed: 28.1 °C")
    expect(chip?.lines[1]).toContain("out of date")
  })

  it("names a daily maximum as such", () => {
    const chip = observedChip(hit(MET_STATIONS, { shown: 31, reading: "daily", observedAt: null }))
    expect(chip?.lines[0]).toBe("Observed, daily maximum: 31.0 °C")
  })

  it("says how a rain total was reached", () => {
    const chip = observedChip(
      hit(RAIN_GAUGE_POINTS, {
        name: "Ampara",
        rainMm: 77.8,
        method: "sum_gauge",
        windowEnd: null,
      }),
    )
    expect(chip?.lines[0]).toBe("Rain, 24 h: 77.8 mm")
    expect(chip?.lines[1]).toContain("river gauge")
  })

  it("leaves other layers to the caller", () => {
    expect(observedChip(hit("other", {}))).toBeNull()
  })
})
