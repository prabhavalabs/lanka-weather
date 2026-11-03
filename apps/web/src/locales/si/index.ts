import type { Translation } from "@/lib/i18n"
import type { en } from "../en"
import { common } from "./common"
import { install } from "./install"
import { pwa } from "./pwa"

export const si: Translation<typeof en> = { common, install, pwa }
