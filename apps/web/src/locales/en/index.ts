/**
 * The English catalogue, one namespace per file. Add a namespace here and in
 * `si/index.ts` and `ta/index.ts` together.
 */
import { common } from "./common"
import { install } from "./install"
import { pwa } from "./pwa"

export const en = { common, install, pwa }
