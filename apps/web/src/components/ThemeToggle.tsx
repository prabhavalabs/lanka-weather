import { MonitorIcon, MoonIcon, SunIcon } from "lucide-react"

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { isThemeMode, useTheme, type ThemeMode } from "@/lib/theme"
import { cn } from "@/lib/utils"

const options: { mode: ThemeMode; label: string; Icon: typeof SunIcon }[] = [
  { mode: "light", label: "Light", Icon: SunIcon },
  { mode: "dark", label: "Dark", Icon: MoonIcon },
  { mode: "system", label: "Same as device", Icon: MonitorIcon },
]

/** Light, dark or the device's setting; the choice is saved in this browser. */
export function ThemeToggle({
  className,
  showLabels = false,
}: {
  className?: string
  /** Shows words beside the icons, e.g. in a settings sheet. */
  showLabels?: boolean
}) {
  const { mode, setMode } = useTheme()

  return (
    <ToggleGroup
      type="single"
      size="sm"
      value={mode}
      onValueChange={(value) => {
        // Radix sends "" when the pressed item is pressed again; keep the choice.
        if (isThemeMode(value)) setMode(value)
      }}
      aria-label="Appearance"
      className={cn(showLabels && "w-full", className)}
    >
      {options.map(({ mode: option, label, Icon }) =>
        showLabels ? (
          <ToggleGroupItem key={option} value={option} className="flex-1" aria-label={label}>
            <Icon />
            <span>{option === "system" ? "Auto" : label}</span>
          </ToggleGroupItem>
        ) : (
          <Tooltip key={option}>
            <TooltipTrigger asChild>
              <ToggleGroupItem value={option} aria-label={label}>
                <Icon />
              </ToggleGroupItem>
            </TooltipTrigger>
            <TooltipContent side="bottom">{label}</TooltipContent>
          </Tooltip>
        ),
      )}
    </ToggleGroup>
  )
}
