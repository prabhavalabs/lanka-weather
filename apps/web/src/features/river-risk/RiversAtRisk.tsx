import { CheckIcon } from "lucide-react"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { Badge } from "@/components/ui/badge"
import { riskBadge, riskHeadline, riskLabels, riversAtRisk } from "./present"
import { RiverRiskLegend } from "./RiverRiskLegend"
import type { RiskDocument } from "./types"

interface RiversAtRiskProps {
  data: RiskDocument | undefined
  onSelect: (id: string) => void
}

/**
 * Rivers at risk, ranked, under the Rivers layer. With none, it says so calmly
 * with the data's age, and never that rivers are safe: it reports what gauges
 * read, and how many gauges that is.
 */
export function RiversAtRisk({ data, onSelect }: RiversAtRiskProps) {
  if (!data) return <RiverRiskLegend />
  const headline = riskHeadline(data)
  const ranked = riversAtRisk(data.rivers)
  const items: GaugeListItem[] = ranked.map((river) => ({
    id: river.id,
    title: river.name,
    subtitle: river.reasons[0]?.text,
    badge: { variant: riskBadge[river.level], label: riskLabels[river.level] },
    onSelect: () => onSelect(river.id),
  }))
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-1.5" role="status">
        <div className="flex items-center justify-between gap-2">
          <p className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            {headline.calm && (
              <CheckIcon aria-hidden="true" className="size-3.5 shrink-0 text-status-normal" />
            )}
            {headline.title}
          </p>
          <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem]">
            Estimate
          </Badge>
        </div>
        <p className="numeric text-xs text-muted-foreground">{headline.detail}</p>
      </div>
      {ranked.length > 0 && <GaugeList title="Rivers at risk" items={items} />}
      <RiverRiskLegend />
      <p className="text-xs text-muted-foreground">
        Estimated from gauges, Google&apos;s forecast and forecast rain. Follow official warnings
        from the Irrigation Department, the Department of Meteorology and the DMC.
      </p>
    </div>
  )
}
