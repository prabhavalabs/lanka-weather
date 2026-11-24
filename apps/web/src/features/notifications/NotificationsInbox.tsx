import {
  ArrowLeftIcon,
  BellIcon,
  BellRingIcon,
  CloudOffIcon,
  LoaderCircleIcon,
  MapIcon,
  SettingsIcon,
  ShieldCheckIcon,
  XIcon,
} from "lucide-react"
import { useEffect, useRef } from "react"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { enable, useAlerts } from "@/features/alerts/store"
import type { Severity } from "@/features/pwa/push"
import { formatAge, formatRecent } from "@/lib/format"
import { useLocale, useT } from "@/lib/i18n"
import {
  type InboxEvent,
  type InboxItem,
  isUnread,
  linkTarget,
  markSeen,
  noticeAreas,
  severityBadge,
  severityColor,
} from "./present"
import { setMarks, showNotice, showTab, useInboxMarks, useInboxView } from "./state"
import type { Inbox } from "./useInbox"

interface InboxProps {
  inbox: Inbox
  /** Opens a notice's map link in the app, closing the panel. */
  onShowOnMap: (url: string) => void
}

function SeverityBadge({ severity }: { severity: Severity }) {
  const t = useT()
  return (
    <Badge variant={severityBadge[severity]} className="h-5 px-2">
      {t(`notifications.severity.${severity}`)}
    </Badge>
  )
}

function When({ iso, prefix }: { iso: string; prefix?: "sentAt" | "issuedAt" | "detectedAt" }) {
  const t = useT()
  const text = formatRecent(iso)
  return (
    <time dateTime={iso} className="numeric">
      {prefix ? t(`notifications.${prefix}`, { time: text }) : text}
      {t("common.separator")}
      {formatAge(iso)}
    </time>
  )
}

function Officialness({ official }: { official: boolean }) {
  const t = useT()
  return official ? (
    <Badge variant="brand" className="h-5 px-2">
      <ShieldCheckIcon aria-hidden="true" />
      {t("notifications.official")}
    </Badge>
  ) : (
    <Badge variant="outline" className="h-5 px-2 whitespace-normal">
      {t("notifications.estimate")}
    </Badge>
  )
}

function ShowOnMap({ url, onShowOnMap }: { url: string; onShowOnMap: (url: string) => void }) {
  const t = useT()
  if (!linkTarget(url).selection) return null
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      className="self-start"
      onClick={() => onShowOnMap(url)}
    >
      <MapIcon />
      {t("notifications.showOnMap")}
    </Button>
  )
}

function Empty({ cleared }: { cleared: boolean }) {
  const t = useT()
  const locale = useLocale()
  const alerts = useAlerts()
  const off = alerts.status !== "on" && alerts.status !== "checking"
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border px-4 py-8 text-center">
      <BellIcon className="size-6 text-muted-foreground" aria-hidden="true" />
      <p className="font-medium">{t("notifications.empty.title")}</p>
      <p className="text-sm text-muted-foreground">
        {cleared ? t("notifications.empty.cleared") : t("notifications.empty.body")}
      </p>
      {off && (
        <>
          <p className="text-sm text-muted-foreground">{t("notifications.empty.off")}</p>
          <div className="flex flex-wrap justify-center gap-2">
            {alerts.status === "off" && (
              <Button
                type="button"
                size="sm"
                disabled={alerts.busy}
                onClick={() => void enable(locale)}
              >
                {alerts.busy ? <LoaderCircleIcon className="animate-spin" /> : <BellRingIcon />}
                {t("notifications.empty.turnOn")}
              </Button>
            )}
            <Button type="button" variant="secondary" size="sm" onClick={() => showTab("settings")}>
              <SettingsIcon />
              {t("notifications.empty.settings")}
            </Button>
          </div>
          {alerts.message && <p className="text-sm">{t(alerts.message)}</p>}
        </>
      )}
    </div>
  )
}

function Row({
  item,
  unread,
  onDismiss,
}: {
  item: InboxItem
  unread: boolean
  onDismiss: () => void
}) {
  const t = useT()
  const server = item.server
  return (
    <li className="relative flex items-stretch gap-1 rounded-2xl border border-border bg-card/60 transition-colors hover:bg-muted/60">
      <span
        aria-hidden="true"
        className="my-3 ml-3 w-1 shrink-0 rounded-full"
        style={{ background: severityColor[item.severity] }}
      />
      <button
        type="button"
        onClick={() => showNotice(item.id)}
        className="flex min-w-0 flex-1 flex-col gap-1 rounded-2xl py-3 pr-1 pl-2 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <span className="flex flex-wrap items-center gap-1.5">
          <SeverityBadge severity={item.severity} />
          {unread && (
            <span className="inline-flex items-center gap-1 text-xs font-medium text-brand-text">
              <span className="size-1.5 rounded-full bg-brand" aria-hidden="true" />
              {t("notifications.new")}
            </span>
          )}
        </span>
        <span className="line-clamp-2 text-sm leading-snug font-medium break-words text-foreground">
          {item.title}
        </span>
        {item.body && (
          <span className="line-clamp-3 text-xs break-words text-muted-foreground">
            {item.body}
          </span>
        )}
        <span className="text-[0.6875rem] text-muted-foreground">
          <When iso={item.time} />
        </span>
        {server && (
          <span className="flex flex-wrap items-center gap-1.5 text-[0.6875rem] text-muted-foreground">
            {server.source && <span>{server.source}</span>}
            <Officialness official={server.official} />
          </span>
        )}
      </button>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="mt-2 mr-2"
        aria-label={t("notifications.dismissNamed", { title: item.title })}
        title={t("notifications.dismiss")}
        onClick={onDismiss}
      >
        <XIcon />
      </Button>
    </li>
  )
}

function EventBlock({
  event,
  onShowOnMap,
}: {
  event: InboxEvent
  onShowOnMap: (url: string) => void
}) {
  return (
    <li className="flex flex-col gap-2 rounded-2xl border border-border p-3">
      <div className="flex items-start justify-between gap-2">
        <p className="min-w-0 text-sm leading-snug font-medium break-words">{event.title}</p>
        <SeverityBadge severity={event.severity} />
      </div>
      {event.body && (
        <p className="text-sm break-words whitespace-pre-line text-foreground">{event.body}</p>
      )}
      <p className="flex flex-col gap-0.5 text-xs text-muted-foreground">
        {event.issuedAt && <When iso={event.issuedAt} prefix="issuedAt" />}
        {event.detectedAt && <When iso={event.detectedAt} prefix="detectedAt" />}
      </p>
      {event.areas.length > 0 && (
        <p className="text-xs text-muted-foreground">
          {event.areas.map((area) => area.name).join(", ")}
        </p>
      )}
      <ShowOnMap url={event.url} onShowOnMap={onShowOnMap} />
    </li>
  )
}

function Detail({
  item,
  pending,
  onShowOnMap,
  onDismiss,
}: {
  item: InboxItem | undefined
  pending: boolean
  onShowOnMap: (url: string) => void
  onDismiss: (id: string) => void
}) {
  const t = useT()
  const heading = useRef<HTMLHeadingElement>(null)
  // Focus moves to the notice, which also brings the top of it into view.
  useEffect(() => {
    heading.current?.focus()
  }, [item?.id])

  const back = (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="-ml-2 self-start"
      onClick={() => showNotice(null)}
    >
      <ArrowLeftIcon />
      {t("notifications.back")}
    </Button>
  )

  if (!item) {
    return (
      <div className="flex flex-col gap-4">
        {back}
        <p role="status" className="flex items-center gap-2 text-sm text-muted-foreground">
          {pending && <LoaderCircleIcon className="size-4 animate-spin" />}
          {pending ? t("notifications.loading") : t("notifications.notFound")}
        </p>
      </div>
    )
  }

  const server = item.server
  const areas = server ? noticeAreas(server) : []
  return (
    <article className="flex flex-col gap-5" aria-labelledby="notice-title">
      {back}
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-1.5">
          <SeverityBadge severity={item.severity} />
          {server && <Officialness official={server.official} />}
        </div>
        <h3
          id="notice-title"
          ref={heading}
          tabIndex={-1}
          className="text-lg leading-snug font-semibold break-words outline-none"
        >
          {item.title}
        </h3>
        <p className="text-xs text-muted-foreground">
          <When iso={item.time} prefix="sentAt" />
        </p>
        {server?.source && (
          <p className="text-xs text-muted-foreground">
            {t("common.source")}: <span className="text-foreground">{server.source}</span>
          </p>
        )}
      </header>

      {server ? (
        <>
          <section className="flex flex-col gap-2">
            <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
              {t("notifications.areas")}
            </h4>
            {server.national || areas.length === 0 ? (
              <Badge variant="chip" className="h-7">
                {t("notifications.wholeCountry")}
              </Badge>
            ) : (
              <ul className="flex flex-wrap gap-1.5">
                {areas.map((area) => (
                  <li key={area.id}>
                    <Badge variant="chip" className="h-7 text-foreground">
                      {area.name}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}
          </section>
          <section className="flex flex-col gap-2">
            <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
              {t("notifications.details")}
            </h4>
            {server.events.length > 0 ? (
              <ul className="flex flex-col gap-2">
                {server.events.map((event) => (
                  <EventBlock key={event.key} event={event} onShowOnMap={onShowOnMap} />
                ))}
              </ul>
            ) : (
              <>
                {server.body && (
                  <p className="text-sm break-words whitespace-pre-line">{server.body}</p>
                )}
                <ShowOnMap url={server.url} onShowOnMap={onShowOnMap} />
              </>
            )}
          </section>
          {!server.official && (
            <p className="rounded-xl border border-border bg-muted/50 p-3 text-xs text-muted-foreground">
              {t("notifications.estimateNote")}
            </p>
          )}
        </>
      ) : (
        <section className="flex flex-col gap-3">
          {item.body && <p className="text-sm break-words whitespace-pre-line">{item.body}</p>}
          <ShowOnMap url={item.url} onShowOnMap={onShowOnMap} />
          <p className="flex items-start gap-2 text-xs text-muted-foreground">
            <CloudOffIcon className="mt-px size-3.5 shrink-0" aria-hidden="true" />
            {t("notifications.savedCopy")}
          </p>
        </section>
      )}

      <div className="border-t border-border pt-4">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => {
            onDismiss(item.id)
            showNotice(null)
          }}
        >
          <XIcon />
          {t("notifications.dismiss")}
        </Button>
      </div>
    </article>
  )
}

/** The list of alerts this device was sent, or one of them in detail. */
export function NotificationsInbox({ inbox, onShowOnMap }: InboxProps) {
  const t = useT()
  const view = useInboxView()
  const marks = useInboxMarks()
  const alerts = useAlerts()

  // What is listed while the panel is open has been seen.
  const { visible } = inbox
  useEffect(() => {
    if (view.open) setMarks((current) => markSeen(current, visible, Date.now()))
  }, [view.open, visible])

  if (view.noticeId !== null) {
    return (
      <Detail
        item={inbox.all.find((item) => item.id === view.noticeId)}
        pending={inbox.pending}
        onShowOnMap={onShowOnMap}
        onDismiss={inbox.dismiss}
      />
    )
  }

  const offline = typeof navigator !== "undefined" && !navigator.onLine
  const cleared = marks.clearedBefore !== null || Object.keys(marks.cleared).length > 0
  const fresh = visible.filter((item) => isUnread(item, view.seenBefore)).length
  return (
    <div className="flex flex-col gap-3">
      {alerts.status === "on" && (offline || inbox.failed) && (
        <p
          role="status"
          className="flex items-start gap-2 rounded-xl border border-border bg-muted/50 p-3 text-xs text-muted-foreground"
        >
          <CloudOffIcon className="mt-px size-3.5 shrink-0" aria-hidden="true" />
          {offline ? t("notifications.offline") : t("notifications.failed")}
        </p>
      )}
      {visible.length === 0 ? (
        inbox.pending ? (
          <p role="status" className="flex items-center gap-2 text-sm text-muted-foreground">
            <LoaderCircleIcon className="size-4 animate-spin" />
            {t("common.loading")}
          </p>
        ) : (
          <Empty cleared={cleared} />
        )
      ) : (
        <>
          <div className="flex items-center justify-between gap-2">
            <p className="text-xs text-muted-foreground">
              {t("notifications.count", { count: visible.length })}
              {fresh > 0 && (
                <>
                  {t("common.separator")}
                  {t("notifications.newCount", { count: fresh })}
                </>
              )}
            </p>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                inbox.dismissAll()
                toast(t("notifications.clearAllDone"), { id: "inbox-cleared" })
              }}
            >
              {t("notifications.clearAll")}
            </Button>
          </div>
          <ul className="flex flex-col gap-2">
            {visible.map((item) => (
              <Row
                key={item.id}
                item={item}
                unread={isUnread(item, view.seenBefore)}
                onDismiss={() => inbox.dismiss(item.id)}
              />
            ))}
          </ul>
        </>
      )}
      {visible.length > 0 && (
        <p className="pt-2 text-xs text-muted-foreground">{t("notifications.empty.body")}</p>
      )}
    </div>
  )
}
