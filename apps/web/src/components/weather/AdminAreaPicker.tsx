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
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

import type { AdminArea, AdminLevel } from "./types"

interface AdminAreaPickerProps {
  level: AdminLevel
  /** Areas at the level. */
  areas: AdminArea[]
  selectedAreaId?: string | null
  onAreaChange: (id: string | null) => void
  className?: string
}

/** Chooses one area at an administrative level, with type-ahead. */
export function AdminAreaPicker({
  level,
  areas,
  selectedAreaId = null,
  onAreaChange,
  className,
}: AdminAreaPickerProps) {
  const t = useT()
  const [open, setOpen] = React.useState(false)
  const selected = areas.find((area) => area.id === selectedAreaId)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          role="combobox"
          aria-expanded={open}
          aria-label={t(`weather.areaPicker.choose.${level}`)}
          className={cn("min-w-40 justify-between", className)}
        >
          <span className={cn("truncate", !selected && "text-muted-foreground")}>
            {selected?.name ?? t(`weather.areaPicker.all.${level}`)}
          </span>
          <ChevronsUpDownIcon className="text-muted-foreground" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-64 p-0" align="start">
        <Command>
          <CommandInput placeholder={t(`weather.areaPicker.find.${level}`)} />
          <CommandList>
            <CommandEmpty>{t("weather.areaPicker.noMatch")}</CommandEmpty>
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
                  <span className="min-w-0 flex-1 truncate">{area.name}</span>
                  {area.parentName && (
                    <span className="min-w-0 truncate text-xs text-muted-foreground">
                      {area.parentName}
                    </span>
                  )}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
