/** The push API (`/api/push`, docs/api/push-alerts.md). Only our own API is called. */
import { ApiError, apiGet } from "@/lib/api"
import type { Locale } from "@/lib/i18n"
import type { Preferences } from "./present"

export interface PushKey {
  enabled: boolean
  publicKey?: string
}

export interface SubscriptionView {
  locale: Locale
  preferences: Preferences
  createdAt: string
  updatedAt: string
  lastSuccessAt: string | null
}

export function getPushKey(signal?: AbortSignal): Promise<PushKey> {
  return apiGet<PushKey>("/api/push/key", signal)
}

async function send<T>(method: string, path: string, body: unknown): Promise<T> {
  const response = await fetch(path, {
    method,
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  const envelope = (await response.json().catch(() => ({}))) as {
    data?: T
    error?: { code: string; message: string }
  }
  if (!response.ok || envelope.error || envelope.data === undefined) {
    throw new ApiError(
      response.status,
      envelope.error?.code ?? "unexpected_response",
      envelope.error?.message ?? `Request failed with status ${response.status}`,
    )
  }
  return envelope.data
}

/** The browser's subscription with the server's preferences shape. */
function toServerPreferences(preferences: Preferences) {
  return {
    ...preferences,
    areas: { ...preferences.areas, near: preferences.areas.near ?? undefined },
  }
}

export function subscribe(
  subscription: PushSubscriptionJSON,
  preferences: Preferences,
  locale: Locale,
) {
  return send<SubscriptionView>("POST", "/api/push/subscriptions", {
    subscription,
    preferences: toServerPreferences(preferences),
    locale,
  })
}

export function updatePreferences(
  subscription: PushSubscriptionJSON,
  preferences: Preferences,
  locale: Locale,
) {
  return send<SubscriptionView>("PUT", "/api/push/subscriptions", {
    endpoint: subscription.endpoint,
    auth: subscription.keys?.auth,
    preferences: toServerPreferences(preferences),
    locale,
  })
}

export function unsubscribe(subscription: PushSubscriptionJSON) {
  return send<{ deleted: boolean }>("DELETE", "/api/push/subscriptions", {
    endpoint: subscription.endpoint,
    auth: subscription.keys?.auth,
  })
}

export function sendTest(subscription: PushSubscriptionJSON) {
  return send<{ sent: boolean }>("POST", "/api/push/test", {
    endpoint: subscription.endpoint,
    auth: subscription.keys?.auth,
  })
}

/**
 * The notices sent to this subscription in the last 30 days, newest first,
 * written in `locale`. POST because the body carries the subscription's
 * auth secret. The answer is checked by features/notifications/present.ts.
 */
export function getInbox(subscription: PushSubscriptionJSON, locale: Locale) {
  return send<{ notices: unknown }>("POST", "/api/push/inbox", {
    endpoint: subscription.endpoint,
    auth: subscription.keys?.auth,
    locale,
  })
}
