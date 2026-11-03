import type { Translation } from "@/lib/i18n"
import type { en } from "../en"
import { common } from "./common"
import { format } from "./format"
import { terms } from "./terms"
import { app } from "./app"
import { layers } from "./layers"
import { map } from "./map"
import { floods } from "./floods"
import { rivers } from "./rivers"
import { reservoirs } from "./reservoirs"
import { warnings } from "./warnings"
import { cyclones } from "./cyclones"
import { earthquakes } from "./earthquakes"
import { weather } from "./weather"
import { grid } from "./grid"
import { location } from "./location"
import { regions } from "./regions"
import { search } from "./search"
import { overview } from "./overview"
import { stations } from "./stations"
import { rain } from "./rain"
import { environment } from "./environment"
import { install } from "./install"
import { pwa } from "./pwa"

export const ta: Translation<typeof en> = {
  common,
  format,
  terms,
  app,
  layers,
  map,
  floods,
  rivers,
  reservoirs,
  warnings,
  cyclones,
  earthquakes,
  weather,
  grid,
  location,
  regions,
  search,
  overview,
  stations,
  rain,
  environment,
  install,
  pwa,
}
