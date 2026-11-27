import {
  AlertTriangleIcon,
  BookOpenIcon,
  ExternalLinkIcon,
  CodeIcon,
  MessageSquareIcon,
  PhoneIcon,
} from "lucide-react"
import type { ReactNode } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { locales, useLocale, useT } from "@/lib/i18n"
import {
  hydroshedsCitation,
  issuesUrl,
  repositoryUrl,
  sourceGroups,
  sources,
  type Link,
} from "./sources"
import { closeAbout, useAboutOpen } from "./state"

/** When the page was last checked against the sources and the code. */
const reviewedOn = "2026-11-27"
const catalogueUrl = `${repositoryUrl}/blob/main/docs/data-sources.md`

function External({ href, children }: { href: string; children: ReactNode }) {
  const t = useT()
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="inline-flex items-center gap-1 rounded-sm font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
    >
      {children}
      <ExternalLinkIcon className="size-3 shrink-0" aria-hidden="true" />
      <span className="sr-only">{t("warnings.newTab")}</span>
    </a>
  )
}

function Section({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section id={id} aria-labelledby={`${id}-title`} className="flex scroll-mt-4 flex-col gap-3">
      <h2 id={`${id}-title`} className="text-base font-semibold">
        {title}
      </h2>
      {children}
    </section>
  )
}

function Points({ items }: { items: string[] }) {
  return (
    <ul className="flex list-disc flex-col gap-2 pl-5 text-sm leading-relaxed text-foreground/90 marker:text-muted-foreground">
      {items.map((item) => (
        <li key={item}>{item}</li>
      ))}
    </ul>
  )
}

function ViaLinks({ links }: { links: Link[] }) {
  return (
    <span className="inline-flex flex-wrap gap-x-3 gap-y-1">
      {links.map((link) => (
        <span key={link.url + link.name} lang="en">
          <External href={link.url}>{link.name}</External>
        </span>
      ))}
    </span>
  )
}

/**
 * Where everything on the map comes from, how it is handled, thanks to the
 * agencies and projects it is built on, the privacy notes and the
 * disclaimer: one page, opened from the sidebar and the phone menu, and at
 * `/?about`.
 */
export function AboutPage() {
  const open = useAboutOpen()
  const t = useT()
  const locale = useLocale()
  const reviewed = new Intl.DateTimeFormat(locales[locale].tag, { dateStyle: "long" }).format(
    new Date(`${reviewedOn}T12:00:00+05:30`),
  )

  const contents = [
    { id: "about-how", label: t("about.how.heading") },
    { id: "about-sources", label: t("about.sources.heading") },
    { id: "about-privacy", label: t("about.privacy.heading") },
    { id: "about-legal", label: t("about.legal.heading") },
    { id: "about-contact", label: t("about.contact.heading") },
  ]

  return (
    <Dialog open={open} onOpenChange={(next) => !next && closeAbout()}>
      <DialogContent
        className="flex h-[calc(100dvh-1rem)] max-h-none w-[calc(100vw-1rem)] max-w-none flex-col gap-0 p-0 sm:h-[calc(100dvh-3rem)] sm:max-w-3xl"
        aria-describedby={undefined}
      >
        <DialogHeader className="gap-1 border-b border-border px-5 pt-5 pb-4 pr-12 text-left sm:px-8">
          <DialogTitle className="text-lg">{t("about.title")}</DialogTitle>
          <DialogDescription>{t("about.subtitle")}</DialogDescription>
        </DialogHeader>
        <ScrollArea
          type="auto"
          className="flex min-h-0 flex-1 flex-col"
          viewportClassName="min-h-0"
        >
          <div className="flex flex-col gap-8 px-5 py-6 sm:px-8">
            {/* What matters most comes first: this is not where official warnings live. */}
            <div
              role="note"
              className="flex flex-col gap-3 rounded-2xl border border-brand bg-brand-soft/40 p-4 sm:p-5"
            >
              <p className="flex items-center gap-2 text-sm font-semibold">
                <AlertTriangleIcon className="size-4 shrink-0 text-brand-text" aria-hidden="true" />
                {t("about.notOfficial.heading")}
              </p>
              <p className="text-sm leading-relaxed">{t("about.notOfficial.body")}</p>
              <Points
                items={[
                  t("about.notOfficial.official"),
                  t("about.notOfficial.estimates"),
                  t("about.notOfficial.stale"),
                  t("about.notOfficial.rely"),
                ]}
              />
              <p className="flex items-center gap-2 text-sm font-semibold">
                <PhoneIcon className="size-4 shrink-0" aria-hidden="true" />
                {t("about.notOfficial.emergency")}
              </p>
            </div>

            <nav aria-label={t("about.contents")}>
              <ul className="flex flex-wrap gap-1.5">
                {contents.map((item) => (
                  <li key={item.id}>
                    <a
                      href={`#${item.id}`}
                      onClick={(event) => {
                        event.preventDefault()
                        document.getElementById(item.id)?.scrollIntoView({ block: "start" })
                      }}
                      className="inline-flex h-8 items-center rounded-full border border-border px-3 text-xs font-medium text-muted-foreground outline-none hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      {item.label}
                    </a>
                  </li>
                ))}
              </ul>
            </nav>

            <Section id="about-how" title={t("about.how.heading")}>
              <Points
                items={[
                  t("about.how.collect"),
                  t("about.how.record"),
                  t("about.how.official"),
                  t("about.how.noAi"),
                  t("about.how.estimates"),
                  t("about.how.languages"),
                ]}
              />
            </Section>

            <Section id="about-sources" title={t("about.sources.heading")}>
              <p className="text-sm leading-relaxed text-foreground/90">
                {t("about.sources.intro")}
              </p>
              {sourceGroups.map((group) => (
                <div key={group} className="flex flex-col gap-2">
                  <h3 className="pt-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
                    {t(`about.sources.groups.${group}`)}
                  </h3>
                  <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-2xl border border-border">
                    {sources
                      .filter((source) => source.group === group)
                      .map((source) => (
                        <li key={source.id} className="flex flex-col gap-1.5 px-4 py-3 text-sm">
                          <span lang="en" className="text-[0.9375rem]">
                            <External href={source.url}>{source.owner}</External>
                          </span>
                          <p className="leading-relaxed text-foreground/90">
                            {t(`about.source.${source.id}.use`)}
                          </p>
                          {source.via && (
                            <p className="flex flex-wrap items-baseline gap-x-2 gap-y-1 text-xs text-muted-foreground">
                              <span>{t("about.sources.via")}:</span>
                              <ViaLinks links={source.via} />
                            </p>
                          )}
                          <p className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                            <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem]">
                              {t("about.sources.terms")}
                            </Badge>
                            <span>{t(`about.source.${source.id}.terms`)}</span>
                          </p>
                          {source.id === "hydrosheds" && (
                            <p className="text-xs text-muted-foreground">
                              {t("about.sources.hydroshedsCitation")}:{" "}
                              <span lang="en">{hydroshedsCitation}</span>
                            </p>
                          )}
                        </li>
                      ))}
                  </ul>
                </div>
              ))}
              <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-foreground/90">
                <BookOpenIcon className="size-4 text-muted-foreground" aria-hidden="true" />
                <span>{t("about.sources.full")}</span>
                <External href={catalogueUrl}>{t("about.sources.catalogue")}</External>
              </p>
            </Section>

            <Section id="about-privacy" title={t("about.privacy.heading")}>
              <Points
                items={[
                  t("about.privacy.noAccount"),
                  t("about.privacy.device"),
                  t("about.privacy.location"),
                  t("about.privacy.alerts"),
                  t("about.privacy.thirdParty"),
                ]}
              />
            </Section>

            <Section id="about-legal" title={t("about.legal.heading")}>
              <Points
                items={[
                  t("about.legal.asIs"),
                  t("about.legal.liability"),
                  t("about.legal.thirdParty"),
                  t("about.legal.boundaries"),
                  t("about.legal.changes"),
                ]}
              />
            </Section>

            <Section id="about-contact" title={t("about.contact.heading")}>
              <p className="text-sm leading-relaxed text-foreground/90">
                {t("about.contact.body")}
              </p>
              <div>
                <Button asChild variant="secondary" size="sm">
                  <a href={issuesUrl} target="_blank" rel="noreferrer">
                    <MessageSquareIcon />
                    {t("about.contact.issues")}
                    <span className="sr-only">{t("warnings.newTab")}</span>
                  </a>
                </Button>
              </div>
            </Section>

            <Section id="about-licence" title={t("about.licence.heading")}>
              <p className="text-sm leading-relaxed text-foreground/90">
                {t("about.licence.body")}
              </p>
              <div>
                <Button asChild variant="secondary" size="sm">
                  <a href={repositoryUrl} target="_blank" rel="noreferrer">
                    <CodeIcon />
                    {t("about.licence.code")}
                    <span className="sr-only">{t("warnings.newTab")}</span>
                  </a>
                </Button>
              </div>
            </Section>

            <p className="border-t border-border pt-4 text-xs text-muted-foreground">
              {t("about.reviewed", { date: reviewed })}
            </p>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
