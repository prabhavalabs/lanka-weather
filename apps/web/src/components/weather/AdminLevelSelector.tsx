import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

import { adminLevelLabel, adminLevels } from "./adminLevels"
import type { AdminLevel } from "./types"

interface AdminLevelSelectorProps {
  /** Boundary level drawn on the map. */
  level: AdminLevel
  onLevelChange: (level: AdminLevel) => void
  /** Levels without data yet. */
  disabledLevels?: AdminLevel[]
  className?: string
}

/**
 * Chooses the administrative level drawn on the map. Pair it with
 * AdminAreaPicker to choose one area at that level; the picker is separate
 * so the map's toolbar does not load the command menu.
 */
export function AdminLevelSelector({
  level,
  onLevelChange,
  disabledLevels = [],
  className,
}: AdminLevelSelectorProps) {
  const t = useT()
  return (
    <ToggleGroup
      type="single"
      size="sm"
      value={level}
      onValueChange={(value) => {
        if (value) onLevelChange(value as AdminLevel)
      }}
      aria-label={t("weather.levelSelector.label")}
      // Wraps rather than overflows where the short names run long (Tamil, Sinhala).
      className={cn("max-w-full flex-wrap", className)}
    >
      {adminLevels.map((option) => {
        const label = adminLevelLabel(option)
        return (
          <ToggleGroupItem
            key={option}
            value={option}
            disabled={disabledLevels.includes(option)}
            aria-label={label.long}
            title={label.long}
          >
            {label.short}
          </ToggleGroupItem>
        )
      })}
    </ToggleGroup>
  )
}
