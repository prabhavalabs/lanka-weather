/** The shareable link that opens the install guide: `/?install`, e.g. sent over WhatsApp or SMS. */
export const installParam = "install"

/** The link people can send each other: it opens the install guide. */
export function installLink(origin = window.location.origin): string {
  return `${origin}/?${installParam}`
}

/** Removes `?install` from a URL, keeping everything else. Null when it is not there. */
export function withoutInstallParam(href: string): string | null {
  const url = new URL(href)
  if (!url.searchParams.has(installParam)) return null
  url.searchParams.delete(installParam)
  return `${url.pathname}${url.search}${url.hash}`
}
