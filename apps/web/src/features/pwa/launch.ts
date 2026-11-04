/**
 * The manifest's start URL (`/?source=pwa`) and shortcuts (`&source=shortcut`)
 * mark how the app was opened. The marker is removed at start-up, so it is
 * not carried into links people share.
 */
/** The URL without its launch marker, or null when it has none. */
export function stripLaunchSource(href: string): string | null {
  const url = new URL(href)
  const source = url.searchParams.get("source")
  if (source !== "pwa" && source !== "shortcut") return null
  url.searchParams.delete("source")
  return `${url.pathname}${url.search}${url.hash}`
}
