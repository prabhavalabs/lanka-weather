import type { Translation } from "@/lib/i18n"
import type { en } from "../en"
import { common } from "./common"
import { pwa } from "./pwa"

export const ta: Translation<typeof en> = { common, pwa }
