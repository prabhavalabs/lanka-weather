/**
 * Keeping a selected place in sight. Cards float over the map (the sidebar
 * and detail column on wide screens, the top bar and the bottom sheet on
 * phones), so "in view" means inside the part of the map they leave clear.
 */

export interface Insets {
  top: number
  right: number
  bottom: number
  left: number
}

const rem = 16
/** Phones: the top bar and its margin. */
export const phoneTopInset = 4.75 * rem
/** Wide screens: the floating toolbar row. */
const desktopTopInset = 4 * rem
const gap = 0.75 * rem

/** How much of the screen the phone sheet covers when it first opens. */
export function sheetPeekHeight(viewportHeight: number): number {
  return Math.round(Math.min(viewportHeight * 0.45, 26 * rem))
}

/** The parts of the map covered by cards, for the current layout. */
export function layoutInsets(layout: {
  desktop: boolean
  sidebarCollapsed: boolean
  detailOpen: boolean
  viewportHeight: number
}): Insets {
  if (!layout.desktop) {
    return {
      top: phoneTopInset,
      right: 0,
      bottom: layout.detailOpen ? sheetPeekHeight(layout.viewportHeight) : 0,
      left: 0,
    }
  }
  return {
    top: desktopTopInset,
    right: layout.detailOpen ? 24.5 * rem + gap : 0,
    bottom: 0,
    left: (layout.sidebarCollapsed ? 4.5 * rem : 22.75 * rem) + gap,
  }
}

/**
 * The offset from the map's centre to the centre of the clear area, for
 * MapLibre's `offset` camera option: centring a place with it puts the
 * place in the middle of what the reader can see.
 */
export function clearCentreOffset(width: number, height: number, insets: Insets): [number, number] {
  const x = (insets.left + (width - insets.right)) / 2 - width / 2
  const y = (insets.top + (height - insets.bottom)) / 2 - height / 2
  return [Math.round(x), Math.round(y)]
}

/** Whether a screen point lies in the clear area, at least `margin` pixels from its edges. */
export function inClearArea(
  point: { x: number; y: number },
  width: number,
  height: number,
  insets: Insets,
  margin = 32,
): boolean {
  return (
    point.x >= insets.left + margin &&
    point.x <= width - insets.right - margin &&
    point.y >= insets.top + margin &&
    point.y <= height - insets.bottom - margin
  )
}

/** Padding for fitting bounds into the clear area. */
export function clearPadding(insets: Insets, margin = 24): Insets {
  return {
    top: insets.top + margin,
    right: insets.right + margin,
    bottom: insets.bottom + margin,
    left: insets.left + margin,
  }
}
