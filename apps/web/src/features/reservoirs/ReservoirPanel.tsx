import { ContainerIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { ProgressBar } from "@/components/ProgressBar"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { Template } from "../rivers/Template"
import {
  flowNote,
  flowShort,
  formatReadingDay,
  readingDayAge,
  reservoirStatusLabels,
  reservoirStatusMeanings,
  reservoirSubtitle,
  storageView,
} from "./present"
import { useReservoirHistory } from "./queries"
import { StorageChart } from "./StorageChart"
import type { ReservoirProperties } from "./types"

const historyDays = 60

/** The small print every reservoir panel carries. */
export function ReservoirSources({ reservoir }: { reservoir?: ReservoirProperties }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-1">
      <p>
        <Template
          text={t("reservoirs.sources.source")}
          values={{
            agency: reservoir?.agency ?? t("reservoirs.sources.agencies"),
            link: (
              <a
                href={reservoir?.sourceUrl ?? "https://waterlevels.lk/reservoirs"}
                target="_blank"
                rel="noreferrer"
                className="underline underline-offset-2 hover:text-foreground"
              >
                {t("reservoirs.sources.link")}
              </a>
            ),
          }}
        />
        {reservoir?.mapped && ` ${t("reservoirs.sources.position")}`}
      </p>
      <p>
        {t("terms.notOfficial")} {t("terms.emergency")}
      </p>
    </div>
  )
}

interface ReservoirPanelProps {
  reservoir: ReservoirProperties
  onClose: () => void
  className?: string
}

/** One reservoir: status in the source's words, storage, flow, age and history. */
export function ReservoirPanel({ reservoir, onClose, className }: ReservoirPanelProps) {
  const t = useT()
  const history = useReservoirHistory(reservoir.id, historyDays)
  const storage = storageView(reservoir)
  const note = reservoir.flow ? flowNote(reservoir.flow) : null

  return (
    <Panel
      icon={<ContainerIcon />}
      title={reservoir.name}
      subtitle={reservoirSubtitle(reservoir)}
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("reservoirs.panel.close")}
        >
          <XIcon />
        </Button>
      }
      footer={<ReservoirSources reservoir={reservoir} />}
    >
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Badge variant={reservoir.status}>{reservoirStatusLabels[reservoir.status]}</Badge>
          {reservoir.stale && (
            <Badge variant="chip" className="text-foreground">
              {t("common.outOfDate")}
            </Badge>
          )}
        </div>
        <p className="text-sm text-muted-foreground">{reservoirStatusMeanings[reservoir.status]}</p>
        <StatValue
          label={t("reservoirs.storage.label")}
          value={storage.value}
          unit={storage.value === "–" ? undefined : "%"}
          caption={storage.caption}
        />
        {storage.current && reservoir.storagePercent !== null && (
          <ProgressBar
            value={Math.min(1, reservoir.storagePercent / 100)}
            label={t("reservoirs.storage.bar")}
          />
        )}
      </div>

      <LabeledRows>
        {reservoir.river && (
          <LabeledRow label={t("reservoirs.panel.river")} value={reservoir.river} numeric={false} />
        )}
        <LabeledRow
          label={t("reservoirs.panel.flow")}
          value={reservoir.flow ? flowShort(reservoir.flow) : t("reservoirs.panel.notReported")}
          numeric={Boolean(reservoir.flow)}
        />
        {reservoir.flow?.agency && (
          <LabeledRow
            label={t("reservoirs.panel.flowBy")}
            value={reservoir.flow.agency}
            numeric={false}
          />
        )}
        <LabeledRow
          label={t("reservoirs.panel.readingDay")}
          value={
            reservoir.readingDay
              ? [formatReadingDay(reservoir.readingDay), readingDayAge(reservoir.readingDay)].join(
                  t("common.separator"),
                )
              : t("common.notKnown")
          }
          tone={reservoir.stale ? "warning" : "default"}
        />
        {reservoir.agency && (
          <LabeledRow
            label={t("reservoirs.panel.reportedBy")}
            value={reservoir.agency}
            numeric={false}
          />
        )}
        {!reservoir.mapped && (
          <LabeledRow
            label={t("reservoirs.panel.onMap")}
            value={t("reservoirs.panel.notMapped")}
            numeric={false}
          />
        )}
      </LabeledRows>
      {note && <p className="text-xs text-muted-foreground">{note}</p>}

      <div>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground">
          {t("reservoirs.panel.history", { days: historyDays })}
        </h3>
        {history.isPending && (
          <Skeleton
            className="h-32 w-full rounded-2xl"
            role="status"
            aria-label={t("reservoirs.panel.loadingHistory")}
          />
        )}
        {history.error && (
          <p
            className="rounded-2xl border border-border bg-inset py-6 text-center text-xs text-muted-foreground"
            role="alert"
          >
            {t("reservoirs.panel.historyUnavailable")}
          </p>
        )}
        {history.data && <StorageChart readings={history.data.readings} days={historyDays} />}
      </div>
    </Panel>
  )
}
