/** Where the legend dock sits, as a move from its place at the bottom left. */
export interface Offset {
  x: number
  y: number
}

export const home: Offset = { x: 0, y: 0 }
/** Room kept between the legends and the edge of the window, in pixels. */
const margin = 8

/**
 * Keeps a box inside the window: the offset that moves `box` (drawn at
 * `offset` from its home) by as little as needed to stay `margin` clear of
 * every edge.
 */
export function clampOffset(
  offset: Offset,
  box: { left: number; top: number; right: number; bottom: number },
  view: { width: number; height: number },
): Offset {
  let { x, y } = offset
  if (box.left < margin) x += margin - box.left
  else if (box.right > view.width - margin) x -= box.right - (view.width - margin)
  if (box.top < margin) y += margin - box.top
  else if (box.bottom > view.height - margin) y -= box.bottom - (view.height - margin)
  return { x: Math.round(x), y: Math.round(y) }
}
