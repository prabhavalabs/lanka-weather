/** Messages of the weather field and wind layers. English is the source: it fixes the keys. */
export const grid = {
  /** What the weather field shows, in the layer's controls. */
  variable: {
    temperature: { label: "Temperature", short: "Temp" },
    precipitation: { label: "Rain, this hour", short: "Rain" },
    humidity: { label: "Relative humidity", short: "Humid" },
    cloud: { label: "Cloud cover", short: "Cloud" },
    wind: { label: "Wind speed", short: "Wind" },
  },
  legend: {
    temperature: "Temperature now",
    precipitation: "Rain, this hour",
    light: "Light",
    heavy: "Heavy",
    violent: "Violent",
    humidity: "Relative humidity",
    /** The colours at the two ends of the scale, for screen readers. */
    humidityLow: "sand",
    humidityHigh: "blue",
    cloud: "Cloud cover",
    cloudLow: "clear",
    cloudHigh: "overcast",
    wind: "Wind speed now",
  },
  /** The wind at a point. The direction is a compass point ("NE"). */
  wind: {
    calm: "Calm",
    text: "{kmh} km/h ({knots} kt) from the {direction}",
  },
  /** The wind forecast's time slider below the map. */
  timeline: {
    section: "Wind forecast time",
    play: "Play the wind forecast",
    pause: "Pause the wind forecast",
    thumb: "Forecast time",
    thumbValue: "{time}, {lead}",
    scale: "Particle colours by wind speed in km/h",
    source: "Wind 10 m · NOAA GFS",
    sourceLoading: "Wind 10 m · NOAA GFS · loading…",
    run: "run {time}",
    /** How far ahead a step is: "Now", "+3 h", "+2 d", "+2 d 6 h". */
    now: "Now",
    leadHours: "+{hours} h",
    leadDays: "+{days} d",
    leadDaysHours: "+{days} d {hours} h",
  },
}
