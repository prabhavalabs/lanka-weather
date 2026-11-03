import { useQuery } from "@tanstack/react-query"
import {
  BellOffIcon,
  BellRingIcon,
  CircleAlertIcon,
  LoaderCircleIcon,
  LocateFixedIcon,
  SendIcon,
  SmartphoneIcon,
} from "lucide-react"
import { useEffect, useMemo, useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { openInstallGuide } from "@/features/install"
import { apiGetBare } from "@/lib/api"
import { intlTag, type Locale, type MessageKey, useLocale, useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import {
  browserFamily,
  inSriLanka,
  type Preferences,
  problem,
  radii,
  roundPoint,
  severities,
  shortDistrictName,
  type Topic,
  topics,
} from "./present"
import { choose, disable, enable, refresh, saveNow, test, useAlerts } from "./store"

interface DistrictFeature {
  properties: { id: string; name: string; nameSi?: string | null; nameTa?: string | null }
}

/** District names from our regions API, in the current language. */
function useDistricts(enabled: boolean) {
  return useQuery({
    queryKey: ["regions", "district", "collection"],
    queryFn: ({ signal }) =>
      apiGetBare<{ features: DistrictFeature[] }>("/v1/regions?level=district", signal),
    select: (data) => data.features.map((feature) => feature.properties),
    staleTime: 7 * 24 * 60 * 60 * 1000,
    // Boundaries come with the names; they are not worth keeping offline.
    meta: { persist: false },
    enabled,
  })
}

function districtName(district: DistrictFeature["properties"], locale: Locale): string {
  const name = locale === "si" ? district.nameSi : locale === "ta" ? district.nameTa : null
  return name ? shortDistrictName(name) : district.name
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <h3 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
        {title}
      </h3>
      {children}
    </section>
  )
}

function Notice({
  tone = "neutral",
  title,
  children,
}: {
  tone?: "neutral" | "warning"
  title?: string
  children: React.ReactNode
}) {
  return (
    <div
      role="status"
      className={cn(
        "flex flex-col gap-2 rounded-2xl border p-3 text-sm",
        tone === "warning"
          ? "border-status-alert/50 bg-status-alert/10"
          : "border-border bg-muted/50",
      )}
    >
      {title && <p className="font-medium">{title}</p>}
      {children}
    </div>
  )
}

/** Where, what and how serious. Editable before alerts are on, saved as they change. */
function Choices({ preferences, locale }: { preferences: Preferences; locale: Locale }) {
  const t = useT()
  const [locating, setLocating] = useState(false)
  const [locationError, setLocationError] = useState<MessageKey | null>(null)
  const districts = useDistricts(!preferences.areas.country)
  const sorted = useMemo(
    () =>
      [...(districts.data ?? [])].sort((a, b) =>
        districtName(a, locale).localeCompare(districtName(b, locale), intlTag(locale)),
      ),
    [districts.data, locale],
  )
  const update = (next: Partial<Preferences>) => choose({ ...preferences, ...next }, locale)
  const areas = (next: Partial<Preferences["areas"]>) =>
    update({ areas: { ...preferences.areas, ...next } })
  const toggleTopic = (topic: Topic, on: boolean) =>
    update({
      topics: topics.filter((item) => (item === topic ? on : preferences.topics.includes(item))),
    })

  const locate = () => {
    if (!("geolocation" in navigator)) return setLocationError("alerts.areas.locationFailed")
    setLocating(true)
    setLocationError(null)
    navigator.geolocation.getCurrentPosition(
      (position) => {
        setLocating(false)
        const { latitude, longitude } = position.coords
        if (!inSriLanka(latitude, longitude))
          return setLocationError("alerts.areas.locationOutside")
        areas({
          near: {
            lat: roundPoint(latitude),
            lon: roundPoint(longitude),
            radiusKm: preferences.areas.near?.radiusKm ?? 10,
          },
        })
      },
      (error) => {
        setLocating(false)
        setLocationError(
          error.code === error.PERMISSION_DENIED
            ? "alerts.areas.locationDenied"
            : "alerts.areas.locationFailed",
        )
      },
      { enableHighAccuracy: false, maximumAge: 10 * 60 * 1000, timeout: 15_000 },
    )
  }

  const issue = problem(preferences)
  return (
    <>
      <Section title={t("alerts.areas.title")}>
        <ToggleGroup
          type="single"
          value={preferences.areas.country ? "country" : "chosen"}
          onValueChange={(value) => value && areas({ country: value === "country" })}
          className="w-full"
        >
          <ToggleGroupItem value="country" className="min-w-0 flex-1 whitespace-normal">
            {t("alerts.areas.country")}
          </ToggleGroupItem>
          <ToggleGroupItem value="chosen" className="min-w-0 flex-1 whitespace-normal">
            {t("alerts.areas.chosen")}
          </ToggleGroupItem>
        </ToggleGroup>
        {!preferences.areas.country && (
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <div className="flex items-baseline justify-between gap-2">
                <p className="text-sm font-medium">{t("alerts.areas.districts")}</p>
                <p className="text-xs text-muted-foreground">
                  {t("alerts.areas.districtsCount", { count: preferences.areas.districts.length })}
                </p>
              </div>
              {districts.isPending && (
                <p className="text-sm text-muted-foreground">
                  {t("alerts.areas.districtsLoading")}
                </p>
              )}
              {districts.isError && (
                <p className="text-sm text-destructive">{t("alerts.areas.districtsFailed")}</p>
              )}
              {sorted.length > 0 && (
                <ul className="grid max-h-56 grid-cols-1 gap-1 overflow-y-auto rounded-xl border border-border p-2 min-[24rem]:grid-cols-2">
                  {sorted.map((district) => {
                    const checked = preferences.areas.districts.includes(district.id)
                    const id = `alert-district-${district.id}`
                    return (
                      <li key={district.id}>
                        <label
                          htmlFor={id}
                          className="flex min-h-9 cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm hover:bg-muted"
                        >
                          <Checkbox
                            id={id}
                            checked={checked}
                            onCheckedChange={(value) =>
                              areas({
                                districts: value
                                  ? [...preferences.areas.districts, district.id].sort()
                                  : preferences.areas.districts.filter(
                                      (item) => item !== district.id,
                                    ),
                              })
                            }
                          />
                          <span className="min-w-0 break-words">
                            {districtName(district, locale)}
                          </span>
                        </label>
                      </li>
                    )
                  })}
                </ul>
              )}
            </div>
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between gap-3">
                <Label htmlFor="alert-near-me" className="flex items-center gap-2">
                  <LocateFixedIcon className="size-4" />
                  {t("alerts.areas.nearMe")}
                </Label>
                <Switch
                  id="alert-near-me"
                  checked={preferences.areas.near !== null}
                  disabled={locating}
                  onCheckedChange={(on) => (on ? locate() : areas({ near: null }))}
                />
              </div>
              <p className="text-xs text-muted-foreground">{t("alerts.areas.nearMeHint")}</p>
              {locating && (
                <p className="text-sm text-muted-foreground">{t("alerts.areas.locating")}</p>
              )}
              {locationError && <p className="text-sm text-destructive">{t(locationError)}</p>}
              {preferences.areas.near && (
                <>
                  <p className="text-xs text-muted-foreground">
                    {t("alerts.areas.savedPoint", {
                      lat: preferences.areas.near.lat.toFixed(2),
                      lon: preferences.areas.near.lon.toFixed(2),
                    })}
                  </p>
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm">{t("alerts.areas.radius")}</span>
                    <ToggleGroup
                      type="single"
                      size="sm"
                      value={String(preferences.areas.near.radiusKm)}
                      onValueChange={(value) =>
                        value &&
                        preferences.areas.near &&
                        areas({ near: { ...preferences.areas.near, radiusKm: Number(value) } })
                      }
                    >
                      {radii.map((km) => (
                        <ToggleGroupItem key={km} value={String(km)} className="numeric">
                          {t("alerts.areas.km", { km })}
                        </ToggleGroupItem>
                      ))}
                    </ToggleGroup>
                  </div>
                </>
              )}
            </div>
            {issue === "areas" && (
              <p className="text-sm text-destructive">{t("alerts.areas.needOne")}</p>
            )}
          </div>
        )}
      </Section>

      <Section title={t("alerts.topics.title")}>
        <ul className="flex flex-col gap-1">
          {topics.map((topic) => {
            const id = `alert-topic-${topic}`
            return (
              <li key={topic}>
                <label
                  htmlFor={id}
                  className="flex cursor-pointer items-start gap-3 rounded-xl px-2 py-2 hover:bg-muted"
                >
                  <Checkbox
                    id={id}
                    className="mt-0.5"
                    checked={preferences.topics.includes(topic)}
                    onCheckedChange={(value) => toggleTopic(topic, value === true)}
                  />
                  <span className="flex min-w-0 flex-col gap-0.5">
                    <span className="text-sm font-medium">{t(`alerts.topics.${topic}`)}</span>
                    <span className="text-xs text-muted-foreground">
                      {t(`alerts.topics.${topic}Hint`)}
                    </span>
                  </span>
                </label>
              </li>
            )
          })}
        </ul>
        {issue === "topics" && (
          <p className="text-sm text-destructive">{t("alerts.topics.needOne")}</p>
        )}
      </Section>

      <Section title={t("alerts.severity.title")}>
        <p className="text-xs text-muted-foreground">{t("alerts.severity.hint")}</p>
        <ToggleGroup
          type="single"
          value={preferences.minSeverity}
          onValueChange={(value) =>
            value && update({ minSeverity: value as Preferences["minSeverity"] })
          }
          className="w-full"
        >
          {severities.map((severity) => (
            <ToggleGroupItem
              key={severity}
              value={severity}
              className="min-w-0 flex-1 whitespace-normal"
            >
              {t(`alerts.severity.${severity}`)}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        <p className="text-xs text-muted-foreground">
          {t(`alerts.severity.${preferences.minSeverity}Hint`)}
        </p>
      </Section>
    </>
  )
}

/** The settings: a state for every case a device can be in, then the choices. */
export function AlertsPanel() {
  const t = useT()
  const locale = useLocale()
  const alerts = useAlerts()
  const family = browserFamily(navigator.userAgent)

  useEffect(() => {
    void refresh()
  }, [])

  // Notices are written in the subscriber's language: tell the server when it changes.
  const lastLocale = useRef(locale)
  useEffect(() => {
    if (lastLocale.current === locale) return
    lastLocale.current = locale
    void saveNow(locale)
  }, [locale])

  const status = alerts.status
  const since = alerts.since
    ? new Intl.DateTimeFormat(intlTag(locale), { dateStyle: "medium" }).format(
        new Date(alerts.since),
      )
    : null
  const showChoices = status === "off" || status === "on"
  // The person's own browser first; "generic" covers the rest.
  const deniedOrder = [
    ...(family === "other" ? [] : [family]),
    ...(["chrome", "firefox", "safari", "android"] as const).filter((item) => item !== family),
    "generic" as const,
  ]

  return (
    <div className="flex flex-col gap-6">
      {status === "checking" && (
        <Notice>
          <p className="flex items-center gap-2 text-muted-foreground">
            <LoaderCircleIcon className="size-4 animate-spin" />
            {t("alerts.status.checking")}
          </p>
        </Notice>
      )}
      {status === "unsupported" && (
        <Notice tone="warning">
          <p>{t("alerts.status.unsupported")}</p>
        </Notice>
      )}
      {status === "ios-install" && (
        <Notice tone="warning" title={t("alerts.status.iosInstallTitle")}>
          <p>{t("alerts.status.iosInstall")}</p>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="self-start"
            onClick={() => openInstallGuide({ reason: "alerts" })}
          >
            <SmartphoneIcon />
            {t("alerts.status.iosInstallAction")}
          </Button>
        </Notice>
      )}
      {status === "disabled" && (
        <Notice>
          <p>{t("alerts.status.disabled")}</p>
        </Notice>
      )}
      {status === "denied" && (
        <Notice tone="warning" title={t("alerts.status.deniedTitle")}>
          <p>{t("alerts.status.denied")}</p>
          <ul className="flex list-disc flex-col gap-1 pl-5 text-xs text-muted-foreground">
            {deniedOrder.map((item) => (
              <li key={item}>{t(`alerts.denied.${item}`)}</li>
            ))}
          </ul>
        </Notice>
      )}
      {status === "unavailable" && (
        <Notice tone="warning">
          <p>{t("alerts.status.unavailable")}</p>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="self-start"
            onClick={() => void refresh()}
          >
            {t("alerts.retry")}
          </Button>
        </Notice>
      )}
      {status === "off" && (
        <Notice>
          <p>{t("alerts.status.off")}</p>
          <Button
            type="button"
            className="self-start"
            disabled={alerts.busy || problem(alerts.preferences) !== null}
            onClick={() => void enable(locale)}
          >
            {alerts.busy ? <LoaderCircleIcon className="animate-spin" /> : <BellRingIcon />}
            {alerts.busy ? t("alerts.turningOn") : t("alerts.turnOn")}
          </Button>
        </Notice>
      )}
      {status === "on" && (
        <Notice>
          <p className="flex items-center gap-2 font-medium">
            <BellRingIcon className="size-4 text-brand" />
            {t("alerts.status.on")}
          </p>
          {since && (
            <p className="text-xs text-muted-foreground">
              {t("alerts.status.onSince", { date: since })}
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={alerts.busy}
              onClick={() => void test()}
            >
              <SendIcon />
              {t("alerts.sendTest")}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={alerts.busy}
              onClick={() => void disable()}
            >
              <BellOffIcon />
              {t("alerts.turnOff")}
            </Button>
          </div>
        </Notice>
      )}
      {alerts.message && (
        <p role="status" className="flex items-start gap-2 text-sm">
          <CircleAlertIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          {t(alerts.message)}
        </p>
      )}

      {showChoices && <Choices preferences={alerts.preferences} locale={locale} />}

      {status === "on" && alerts.save !== "idle" && (
        <p
          role="status"
          className={cn(
            "text-xs",
            alerts.save === "failed" ? "text-destructive" : "text-muted-foreground",
          )}
        >
          {t(
            alerts.save === "saving"
              ? "alerts.saving"
              : alerts.save === "saved"
                ? "alerts.saved"
                : "alerts.saveFailed",
          )}
        </p>
      )}

      <div className="flex flex-col gap-2 border-t border-border pt-4 text-xs text-muted-foreground">
        <p>{t("alerts.note")}</p>
        <p>{t("alerts.privacy")}</p>
      </div>
    </div>
  )
}
