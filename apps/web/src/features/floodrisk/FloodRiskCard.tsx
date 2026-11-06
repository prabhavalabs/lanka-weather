import { Badge } from "@/components/ui/badge"
import { formatAge } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { levelKey, riskBadge, signalValue } from "./present"
import type { FloodRiskDistrict } from "./types"

interface FloodRiskCardProps {
  risk: FloodRiskDistrict
  generatedAt: string
  complete: boolean
  className?: string
}

/**
 * A district's combined flood-risk estimate: its level, what could happen,
 * and every signal it rests on with its figures, strongest first. Always
 * called an estimate; level 0 says no two sources agree, never "safe".
 */
export function FloodRiskCard({ risk, generatedAt, complete, className }: FloodRiskCardProps) {
  const t = useT()
  return (
    <section className={cn("flex flex-col gap-1.5", className)} aria-labelledby="flood-risk-title">
      <div className="flex items-center justify-between gap-2">
        <h3 id="flood-risk-title" className="text-xs font-medium text-muted-foreground">
          {t("floodRisk.title")}
        </h3>
        <Badge variant={riskBadge[risk.level]}>{t(levelKey[risk.level])}</Badge>
      </div>
      <div className="flex flex-col gap-2 rounded-xl border border-border bg-card px-3 py-2.5">
        <p className="text-sm text-foreground">
          {risk.level === 0 ? t("floodRisk.none") : t(`floodRisk.scenario.${risk.scenario}`)}
        </p>
        {risk.signals.length > 0 && (
          <ul className="flex flex-col gap-1">
            {risk.signals.map((signal) => {
              const value = signalValue(signal)
              return (
                <li
                  key={signal.family}
                  className="flex items-baseline justify-between gap-3 text-xs"
                >
                  <span className="text-muted-foreground">
                    {t(`floodRisk.signal.${signal.family}`)}
                  </span>
                  <span className="numeric text-right text-foreground">
                    {t(value.key, value.params)}
                  </span>
                </li>
              )
            })}
          </ul>
        )}
      </div>
      <p className="text-[0.6875rem] text-muted-foreground">
        {t("floodRisk.note")}
        {t("common.separator")}
        <span className="numeric">{t("common.readAgo", { age: formatAge(generatedAt) })}</span>
        {!complete && (
          <>
            {t("common.separator")}
            {t("floodRisk.incomplete")}
          </>
        )}
      </p>
    </section>
  )
}
