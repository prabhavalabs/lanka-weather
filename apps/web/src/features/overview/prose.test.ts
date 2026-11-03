import { describe, expect, it } from "vitest"
import { translateIn } from "@/lib/i18n"
import { detailIn, proseIn, statusIn } from "./prose"

describe("the overview's words in Sinhala and Tamil", () => {
  it("keeps the API's English as sent on the English page", () => {
    expect(proseIn("en", "10 reservoirs spilling.")).toEqual({ text: "10 reservoirs spilling." })
    expect(statusIn("en", "1 cyclone active north of Sri Lanka. 10 reservoirs spilling.")).toEqual([
      { text: "1 cyclone active north of Sri Lanka. 10 reservoirs spilling." },
    ])
  })

  it("writes the status again sentence by sentence, with its numbers and directions", () => {
    expect(
      statusIn("si", "1 cyclone active north of Sri Lanka. 10 reservoirs spilling.").map(
        (part) => part.text,
      ),
    ).toEqual([
      `ශ්‍රී ලංකාවට ${translateIn("si", "format.compassLong.N")} දෙසින් සුළි කුණාටුවක් සක්‍රියව පවතී.`,
      "ජලාශ 10ක් වාන් දමයි.",
    ])
    expect(statusIn("ta", "Out of date: river gauges, cyclones.")[0]!.text).toBe(
      "காலாவதியானவை: ஆற்று அளவீட்டு நிலையங்கள், சூறாவளிகள்.",
    )
  })

  it("keeps an estimate an estimate", () => {
    const text = proseIn(
      "si",
      "Estimate: 5 stretches of 233 at watch or above. Nothing measures the 165 not known.",
    ).text
    expect(text).toContain("ඇස්තමේන්තුව")
    expect(text).toContain("233")
    expect(text).toContain("165")
    expect(proseIn("ta", "Estimate by lanka-weather").text).toContain("மதிப்பீடு")
  })

  it("translates official words from terms, and dates from the catalogue", () => {
    expect(proseIn("ta", "Warning: Amber").text).toBe(
      `${translateIn("ta", "terms.bulletinSeverity.warning")}: ${translateIn("ta", "terms.colour.amber")}`,
    )
    expect(proseIn("si", "Level 3 (Red)").text).toBe(translateIn("si", "terms.landslideLevel.l3"))
    expect(
      proseIn("si", "Last report 31 Oct 2026; none currently active as far as we know.").text,
    ).toContain("2026")
  })

  it("marks what it does not know as English", () => {
    expect(proseIn("si", "Warning for strong winds and rough seas")).toEqual({
      text: "Warning for strong winds and rough seas",
      lang: "en",
    })
    expect(detailIn("ta", "1,551 cusecs · 97% full · Kurunegala")).toEqual([
      { text: "1,551 cusecs" },
      { text: "97% நிரம்பியுள்ளது" },
      { text: "Kurunegala", lang: "en" },
    ])
  })
})
