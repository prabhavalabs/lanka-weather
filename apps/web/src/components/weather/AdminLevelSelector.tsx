import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { cn } from "@/lib/utils"

import { adminLevelLabels, adminLevels } from "./adminLevels"
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
  return (
    <ToggleGroup
      type="single"
      size="sm"
      value={level}
      onValueChange={(value) => {
        if (value) onLevelChange(value as AdminLevel)
      }}
      aria-label="Boundary level"
      className={cn(className)}
    >
      {adminLevels.map((option) => (
        <ToggleGroupItem
          key={option}
          value={option}
          disabled={disabledLevels.includes(option)}
          aria-label={adminLevelLabels[option].long}
          title={adminLevelLabels[option].long}
        >
          {adminLevelLabels[option].short}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  )
}
