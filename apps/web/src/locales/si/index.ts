import type { Translation } from "@/lib/i18n"
import type { en } from "../en"
import { alerts } from "./alerts"
import { common } from "./common"

export const si: Translation<typeof en> = { common, alerts }
