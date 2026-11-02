import * as React from "react"
import { ChevronsUpDownIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { cn } from "@/lib/utils"

import type { AdminArea, AdminLevel } from "./types"

const adminLevelLabels: Record<AdminLevel, { short: string; long: string }> = {
  province: { short: "Province", long: "Province" },
  district: { short: "District", long: "District" },
  ds: { short: "DS", long: "Divisional Secretariat division" },
  gn: { short: "GN", long: "Grama Niladhari division" },
}

const levels: AdminLevel[] = ["province", "district", "ds", "gn"]

interface AdminLevelSelectorProps {
  /** Boundary level drawn on the map. */
  level: AdminLevel
  onLevelChange: (level: AdminLevel) => void
  /** Areas at the current level, for the area picker. Omit to hide the picker. */
  areas?: AdminArea[]
  selectedAreaId?: string | null
  onAreaChange?: (id: string | null) => void
  /** Levels without data yet. */
  disabledLevels?: AdminLevel[]
  className?: string
}

/** Chooses the administrative level drawn on the map and, optionally, one area at that level. */
export function AdminLevelSelector({
  level,
  onLevelChange,
  areas,
  selectedAreaId = null,
  onAreaChange,
  disabledLevels = [],
  className,
}: AdminLevelSelectorProps) {
  const [open, setOpen] = React.useState(false)
  const selected = areas?.find((area) => area.id === selectedAreaId)

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      <ToggleGroup
        type="single"
        size="sm"
        value={level}
        onValueChange={(value) => {
          if (value) onLevelChange(value as AdminLevel)
        }}
        aria-label="Boundary level"
      >
        {levels.map((option) => (
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

      {areas && onAreaChange && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              role="combobox"
              aria-expanded={open}
              aria-label={`Choose a ${adminLevelLabels[level].long.toLowerCase()}`}
              className="min-w-40 justify-between"
            >
              <span className={cn("truncate", !selected && "text-muted-foreground")}>
                {selected?.name ?? `All ${adminLevelLabels[level].short.toLowerCase()}s`}
              </span>
              <ChevronsUpDownIcon className="text-muted-foreground" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-64 p-0" align="start">
            <Command>
              <CommandInput placeholder={`Find a ${adminLevelLabels[level].long.toLowerCase()}…`} />
              <CommandList>
                <CommandEmpty>No match.</CommandEmpty>
                <CommandGroup>
                  {areas.map((area) => (
                    <CommandItem
                      key={area.id}
                      data-checked={area.id === selectedAreaId}
                      value={`${area.name} ${area.parentName ?? ""}`}
                      onSelect={() => {
                        onAreaChange(area.id === selectedAreaId ? null : area.id)
                        setOpen(false)
                      }}
                    >
                      <span className="flex-1 truncate">{area.name}</span>
                      {area.parentName && (
                        <span className="text-xs text-muted-foreground">{area.parentName}</span>
                      )}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      )}
    </div>
  )
}
