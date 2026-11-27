import { BellIcon, LandmarkIcon } from "lucide-react"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { badgeText } from "@/features/notifications/present"
import { useT } from "@/lib/i18n"

export type WarningsTab = "official" | "alerts"

const trigger = "h-auto min-h-9 min-w-0 px-2 py-1.5 text-center leading-snug whitespace-normal"

interface WarningsTabsProps {
  tab: WarningsTab
  /** Official warnings and advisories in force. */
  officialCount: number
  /** Alerts this device was sent and has not seen. */
  unread: number
  onTabChange: (tab: WarningsTab) => void
}

/**
 * The two halves of one panel: what the Department of Meteorology, the DMC
 * and NBRO have issued, and the alerts this device was sent about them. One
 * place to look, with the source of each kept apart.
 */
export function WarningsTabs({ tab, officialCount, unread, onTabChange }: WarningsTabsProps) {
  const t = useT()
  const badge = badgeText(unread)
  return (
    <Tabs value={tab} onValueChange={(value) => onTabChange(value as WarningsTab)}>
      {/* Long labels (Sinhala, Tamil) wrap onto a second line rather than run into each other. */}
      <TabsList
        className="w-full items-stretch group-data-horizontal/tabs:h-auto"
        aria-label={t("warnings.tabs.label")}
      >
        <TabsTrigger value="official" className={trigger}>
          <LandmarkIcon />
          <span className="min-w-0 text-balance">{t("warnings.tabs.official")}</span>
          {officialCount > 0 && (
            <span className="numeric rounded-full bg-brand px-1.5 text-[0.625rem] leading-4 font-semibold text-brand-foreground">
              {officialCount}
            </span>
          )}
        </TabsTrigger>
        <TabsTrigger value="alerts" className={trigger}>
          <BellIcon />
          <span className="min-w-0 text-balance">{t("warnings.tabs.alerts")}</span>
          {badge && (
            <span className="numeric rounded-full bg-destructive px-1.5 text-[0.625rem] leading-4 font-semibold text-white">
              {badge}
            </span>
          )}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}
