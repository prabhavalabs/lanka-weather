import { LevelChart } from "./LevelChart"
import {
  formatAge,
  formatLevel,
  formatRate,
  headroom,
  statusColors,
  statusLabels,
  trendArrows,
  trendLabels,
} from "./present"
import { useStationHistory } from "./queries"
import type { StationProperties } from "./types"

const historyHours = 72

interface StationPanelProps {
  station: StationProperties
  onClose: () => void
}

/** Details of one gauging station: current level, thresholds and recent history. */
export function StationPanel({ station, onClose }: StationPanelProps) {
  const history = useStationHistory(station.id, historyHours)
  const room = headroom(station)
  const thresholds = [
    { label: "Alert", value: station.alertLevel, color: statusColors.alert },
    { label: "Minor flood", value: station.minorFloodLevel, color: statusColors.minor_flood },
    { label: "Major flood", value: station.majorFloodLevel, color: statusColors.major_flood },
  ].filter(
    (threshold): threshold is { label: string; value: number; color: string } =>
      threshold.value !== null,
  )

  return (
    <section
      className="pointer-events-auto w-full max-w-sm rounded-xl border border-white/10 bg-slate-950/90 text-slate-100 shadow-xl backdrop-blur"
      aria-labelledby="station-title"
    >
      <header className="flex items-start justify-between gap-3 border-b border-white/10 px-4 py-3">
        <div>
          <h2 id="station-title" className="text-base font-semibold">
            {station.name}
          </h2>
          <p className="text-xs text-slate-400">{station.basin} · river gauge</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="rounded-md px-2 py-1 text-slate-400 hover:bg-white/10 hover:text-slate-100"
          aria-label="Close station details"
        >
          ✕
        </button>
      </header>

      <div className="space-y-3 px-4 py-3">
        <div className="flex items-baseline justify-between">
          <span className="text-3xl font-semibold tabular-nums">
            {formatLevel(station.waterLevel, station.unit)}
          </span>
          <span
            className="rounded-full px-2.5 py-0.5 text-xs font-medium text-slate-950"
            style={{ backgroundColor: statusColors[station.status] }}
          >
            {statusLabels[station.status]}
          </span>
        </div>

        <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
          <dt className="text-slate-400">Trend</dt>
          <dd className="text-right tabular-nums">
            {trendArrows[station.trend]} {trendLabels[station.trend]}
            {station.ratePerHour !== null && ` (${formatRate(station.ratePerHour, station.unit)})`}
          </dd>
          <dt className="text-slate-400">Observed</dt>
          <dd className={`text-right ${station.stale ? "text-amber-300" : ""}`}>
            {formatAge(station.observedAt)}
            {station.stale && " · out of date"}
          </dd>
          {station.rainfallMm !== null && (
            <>
              <dt className="text-slate-400">Rainfall at gauge</dt>
              <dd className="text-right tabular-nums">{station.rainfallMm.toFixed(1)} mm</dd>
            </>
          )}
          {thresholds.map((threshold) => (
            <div key={threshold.label} className="contents">
              <dt className="flex items-center gap-1.5 text-slate-400">
                <span
                  className="inline-block size-2 rounded-full"
                  style={{ backgroundColor: threshold.color }}
                />
                {threshold.label}
              </dt>
              <dd className="text-right tabular-nums">
                {formatLevel(threshold.value, station.unit)}
              </dd>
            </div>
          ))}
        </dl>

        {room && (
          <p className="text-xs text-slate-300">
            {formatLevel(room.remaining, station.unit)} below {room.next} level.
          </p>
        )}
        {thresholds.length === 0 && (
          <p className="text-xs text-slate-400">This station publishes no flood thresholds.</p>
        )}

        <div className="text-sky-400">
          {history.isPending && (
            <p className="py-6 text-center text-xs text-slate-400">Loading history…</p>
          )}
          {history.error && (
            <p className="py-6 text-center text-xs text-amber-300" role="alert">
              History is unavailable.
            </p>
          )}
          {history.data && (
            <LevelChart
              readings={history.data.readings}
              thresholds={thresholds}
              unit={station.unit}
              hours={historyHours}
            />
          )}
        </div>
      </div>

      <footer className="border-t border-white/10 px-4 py-2 text-[11px] text-slate-500">
        Source: Irrigation Department of Sri Lanka. Official flood warnings come from the Irrigation
        Department and the DMC.
      </footer>
    </section>
  )
}
