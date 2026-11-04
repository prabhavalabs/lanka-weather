import { ExternalLinkIcon, TornadoIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { compassPoint, formatAge, formatDayTime, formatNumber } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { formatDistance } from "../earthquakes/present"
import { alertBadge, alertText, evaluationNote } from "./present"
import type { CyclonePosition } from "./types"

interface CyclonePanelProps {
  system: CyclonePosition
  /** The layer has not been checked recently. */
  stale?: boolean
  onClose: () => void
}

/** One cyclone: alert, position, distance from Sri Lanka and the link to GDACS. */
export function CyclonePanel({ system, stale = false, onClose }: CyclonePanelProps) {
  const t = useT()
  return (
    <Panel
      icon={<TornadoIcon />}
      title={system.name}
      subtitle={t("cyclones.panel.subtitle")}
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("cyclones.panel.close")}
        >
          <XIcon />
        </Button>
      }
      footer={<p>{t("cyclones.panel.footer", { age: formatAge(system.fetchedAt) })}</p>}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={alertBadge[system.alertLevel] ?? "unknown"}>
          {alertText(system.alertLevel)}
        </Badge>
        {(stale || system.stale) && (
          <Badge variant="chip" className="text-foreground">
            {t("common.outOfDate")}
          </Badge>
        )}
      </div>
      {system.severityText && (
        <p className="text-sm text-foreground" lang="en">
          {system.severityText}
        </p>
      )}
      <LabeledRows>
        <LabeledRow
          label={t("cyclones.panel.fromCoast")}
          value={formatDistance(system.distanceKm)}
        />
        {system.windKmh !== null && (
          <LabeledRow
            label={t("cyclones.panel.maxWind")}
            value={`${formatNumber(system.windKmh)} km/h`}
          />
        )}
        <LabeledRow
          label={t("cyclones.panel.position")}
          value={t("cyclones.panel.positionValue", {
            lat: Math.abs(system.lat).toFixed(1),
            ns: compassPoint(system.lat >= 0 ? 0 : 180),
            lon: Math.abs(system.lon).toFixed(1),
            ew: compassPoint(system.lon >= 0 ? 90 : 270),
          })}
        />
        <LabeledRow
          label={t("cyclones.panel.advisory")}
          value={`${formatDayTime(system.latestAt)}${t("common.separator")}${formatAge(system.latestAt)}`}
          tone={system.stale ? "warning" : "default"}
        />
        <LabeledRow label={t("cyclones.panel.firstSeen")} value={formatDayTime(system.startedAt)} />
        {system.country && (
          <LabeledRow
            label={t("cyclones.panel.lists")}
            value={<span lang="en">{system.country}</span>}
            numeric={false}
          />
        )}
        {system.advisorySource && (
          <LabeledRow
            label={t("cyclones.panel.trackFrom")}
            value={system.advisorySource}
            numeric={false}
          />
        )}
      </LabeledRows>
      <a
        href={system.reportUrl}
        target="_blank"
        rel="noreferrer"
        className="inline-flex items-center gap-1.5 self-start rounded-sm text-sm font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
      >
        {t("cyclones.panel.report")}
        <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
        <span className="sr-only">{t("cyclones.newTab")}</span>
      </a>
      <p className="text-xs text-muted-foreground">
        {t("cyclones.panel.disclaimer")} {evaluationNote()}
      </p>
    </Panel>
  )
}
