import { DropletsIcon, MapPinIcon, RouteIcon, SquareDashedIcon, TreesIcon } from "lucide-react"
import { useId, type ComponentType } from "react"
import { IconTile } from "@/components/IconTile"
import { Switch } from "@/components/ui/switch"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import {
  basemapGroups,
  setBasemapGroup,
  useBasemapGroups,
  type BasemapGroup,
} from "./basemapGroups"

const icons: Record<BasemapGroup, ComponentType> = {
  places: MapPinIcon,
  waterways: DropletsIcon,
  roads: RouteIcon,
  landcover: TreesIcon,
  boundaries: SquareDashedIcon,
}

/**
 * The basemap's own detail, a switch per group, styled as the layer list
 * above it. Off by default except place names, so the map starts plain.
 */
export function BasemapPanel({ className }: { className?: string }) {
  const t = useT()
  const uid = useId()
  const shown = useBasemapGroups()
  return (
    <section aria-labelledby={`${uid}-heading`} className={className}>
      <h2
        id={`${uid}-heading`}
        className="sticky top-0 z-10 -mx-1 mb-1.5 bg-(--panel-bg,transparent) px-2 py-1.5 text-xs font-medium text-muted-foreground"
      >
        {t("map.basemap.heading")}
      </h2>
      <ul className="flex flex-col gap-1.5">
        {basemapGroups.map((group) => {
          const on = shown.includes(group)
          const Icon = icons[group]
          const switchId = `${uid}-switch-${group}`
          const title = t(`map.basemap.${group}.title`)
          return (
            <li
              key={group}
              className={cn(
                "rounded-2xl border border-transparent transition-colors",
                on && "border-border bg-accent",
              )}
            >
              <div
                className={cn(
                  "flex items-center gap-3 rounded-2xl px-1 py-2",
                  !on && "hover:bg-accent/60",
                )}
              >
                <IconTile>
                  <Icon />
                </IconTile>
                <label htmlFor={switchId} className="min-w-0 flex-1 cursor-pointer">
                  <span className="block truncate text-sm font-medium">{title}</span>
                  <span className="block truncate text-xs text-muted-foreground">
                    {t(`map.basemap.${group}.description`)}
                  </span>
                </label>
                <Switch
                  id={switchId}
                  checked={on}
                  aria-label={title}
                  onCheckedChange={(checked) => setBasemapGroup(group, checked)}
                  className="mr-1"
                />
              </div>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
