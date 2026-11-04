import { CheckIcon } from "lucide-react"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { Badge } from "@/components/ui/badge"
import { useLocale, useT } from "@/lib/i18n"
import {
  reasonBasis,
  reasonText,
  riskBadge,
  riskHeadline,
  riskLabels,
  riverName,
  riversAtRisk,
} from "./present"
import { RiverRiskLegend } from "./RiverRiskLegend"
import type { RiskDocument, RiskReason } from "./types"

/**
 * A reason on one line: in the reader's language when the sentence is one
 * we know, else what it rests on, since the English sentence would be cut.
 */
function reasonLine(reason: RiskReason) {
  const shown = reasonText(reason)
  return shown.english ? reasonBasis(reason.kind) : shown.text
}

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
  const t = useT()
  const locale = useLocale()
  if (!data) return <RiverRiskLegend />
  const headline = riskHeadline(data)
  const ranked = riversAtRisk(data.rivers)
  const items: GaugeListItem[] = ranked.map((river) => ({
    id: river.id,
    title: riverName(river, locale),
    subtitle: river.reasons[0] && reasonLine(river.reasons[0]),
    badge: { variant: riskBadge[river.level], label: riskLabels[river.level] },
    onSelect: () => onSelect(river.id),
  }))
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-1.5" role="status">
        <div className="flex items-center justify-between gap-2">
          <p className="flex min-w-0 items-center gap-1.5 text-sm font-medium text-foreground">
            {headline.calm && (
              <CheckIcon aria-hidden="true" className="size-3.5 shrink-0 text-status-normal" />
            )}
            {headline.title}
          </p>
          <Badge variant="chip" className="h-5 shrink-0 px-1.5 text-[0.6875rem]">
            {t("terms.estimate")}
          </Badge>
        </div>
        <p className="numeric text-xs text-muted-foreground">{headline.detail}</p>
      </div>
      {ranked.length > 0 && <GaugeList title={t("rivers.risk.list.title")} items={items} />}
      <RiverRiskLegend />
      <p className="text-xs text-muted-foreground">{t("rivers.risk.list.footnote")}</p>
    </div>
  )
}
