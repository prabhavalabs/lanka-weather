/** The layer list (features/layers) and the layer catalogue's words (lib/layers). */
export const layers = {
  category: {
    hazard: "Hazards and warnings",
    hydrology: "Rivers and floods",
    weather: "Weather",
    geography: "Geography",
  },
  status: {
    waiting: "Data is still loading · collected shortly",
    error: "Unavailable · retrying shortly",
  },
  /** The age at the end of a layer's one-line status. */
  age: {
    updated: "updated {age}",
    read: "read {age}",
    run: "run {age}",
    image: "image {age}",
    newest: "newest {age}",
    issued: "issued {age}",
  },
}
