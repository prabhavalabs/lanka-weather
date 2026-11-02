import { ChevronDownIcon } from "lucide-react"
import { useId, useState, type ReactNode } from "react"
import { IconTile } from "@/components/IconTile"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { groupByCategory } from "@/lib/layers"
import { cn } from "@/lib/utils"
import { layerIcon } from "./layerIcons"
import { useLayers } from "./useLayers"

interface LayerPanelProps {
  enabled: string[]
  onToggle: (id: string) => void
  /** One-line status per layer id, shown while the layer is on. */
  summaries?: Record<string, string>
  /** Extra content per layer id under a layer that is on, e.g. a status bar or legend. */
  details?: Record<string, ReactNode>
  /** Layers whose data is older than the catalogue allows; marked "Out of date". */
  stale?: Record<string, boolean>
  className?: string
}

/** The layer catalogue grouped by category, with a switch per layer. */
export function LayerPanel({
  enabled,
  onToggle,
  summaries = {},
  details = {},
  stale = {},
  className,
}: LayerPanelProps) {
  const { data, error, isPending } = useLayers()
  // The panel can be mounted twice (sidebar and phone sheet), so ids are unique per instance.
  const uid = useId()
  // Ten layers open at once would fill the list with notes, so a layer's details
  // open on request, and when it is switched on.
  const [opened, setOpened] = useState<string[]>([])
  const setOpen = (id: string, open: boolean) =>
    setOpened((current) =>
      open ? [...new Set([...current, id])] : current.filter((value) => value !== id),
    )

  return (
    <div className={cn("flex flex-col gap-5", className)}>
      {isPending && (
        <div className="flex flex-col gap-3" role="status" aria-label="Loading layers">
          {[0, 1, 2].map((row) => (
            <div key={row} className="flex items-center gap-3">
              <Skeleton className="size-10 rounded-xl" />
              <div className="flex flex-1 flex-col gap-1.5">
                <Skeleton className="h-3.5 w-28" />
                <Skeleton className="h-3 w-40" />
              </div>
            </div>
          ))}
        </div>
      )}
      {error && !data && (
        <p
          className="rounded-xl border border-border bg-inset px-3 py-3 text-sm text-foreground"
          role="alert"
        >
          Layers are unavailable: {error.message}
        </p>
      )}
      {data &&
        groupByCategory(data).map((group) => (
          <section key={group.category} aria-labelledby={`${uid}-group-${group.category}`}>
            <h2
              id={`${uid}-group-${group.category}`}
              className="mb-1.5 px-1 text-xs font-medium text-muted-foreground"
            >
              {group.label}
            </h2>
            <ul className="flex flex-col">
              {group.layers.map((layer) => {
                const planned = layer.status === "planned"
                const on = enabled.includes(layer.id)
                const Icon = layerIcon(layer)
                const switchId = `${uid}-switch-${layer.id}`
                const detail = on && !planned ? details[layer.id] : undefined
                const detailOpen = opened.includes(layer.id)
                const detailId = `${uid}-detail-${layer.id}`
                return (
                  <li
                    key={layer.id}
                    className={cn("rounded-2xl transition-colors", on && !planned && "bg-accent")}
                  >
                    <div
                      className={cn(
                        "flex items-center gap-3 rounded-2xl px-1 py-2",
                        !planned && !on && "hover:bg-accent/60",
                      )}
                      title={layer.description || undefined}
                    >
                      <IconTile className={cn(planned && "text-muted-foreground")}>
                        <Icon />
                      </IconTile>
                      <label
                        htmlFor={switchId}
                        className={cn(
                          "min-w-0 flex-1",
                          planned ? "cursor-default" : "cursor-pointer",
                        )}
                      >
                        <span
                          className={cn(
                            "flex items-center gap-2 text-sm font-medium",
                            planned && "text-muted-foreground",
                          )}
                        >
                          <span className="truncate">{layer.title}</span>
                          {on && !planned && stale[layer.id] && (
                            <Badge
                              variant="chip"
                              className="h-5 px-1.5 text-[0.6875rem] text-foreground"
                            >
                              Out of date
                            </Badge>
                          )}
                        </span>
                        <span className="block truncate text-xs text-muted-foreground">
                          {on && summaries[layer.id] ? summaries[layer.id] : layer.attribution}
                        </span>
                      </label>
                      {detail && (
                        <button
                          type="button"
                          aria-expanded={detailOpen}
                          aria-controls={detailOpen ? detailId : undefined}
                          aria-label={`${detailOpen ? "Hide" : "Show"} details of ${layer.title}`}
                          onClick={() => setOpen(layer.id, !detailOpen)}
                          className="-mr-1 flex size-7 shrink-0 items-center justify-center rounded-full text-muted-foreground outline-none hover:bg-background/60 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
                        >
                          <ChevronDownIcon
                            aria-hidden="true"
                            className={cn(
                              "size-4 transition-transform",
                              detailOpen && "rotate-180",
                            )}
                          />
                        </button>
                      )}
                      {planned ? (
                        <Badge variant="chip" className="mr-1">
                          Soon
                        </Badge>
                      ) : (
                        <Switch
                          id={switchId}
                          checked={on}
                          aria-label={layer.title}
                          onCheckedChange={() => {
                            setOpen(layer.id, !on)
                            onToggle(layer.id)
                          }}
                          className="mr-1"
                        />
                      )}
                    </div>
                    {detail && detailOpen && (
                      <div id={detailId} className="px-3 pt-1 pb-3">
                        {detail}
                      </div>
                    )}
                  </li>
                )
              })}
            </ul>
          </section>
        ))}
    </div>
  )
}
