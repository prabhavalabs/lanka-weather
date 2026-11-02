/** Error returned by the lanka-weather API. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
  }
}

interface Envelope<T> {
  data?: T
  error?: { code: string; message: string }
}

/**
 * Codes the API answers with until the collectors have run (docs/api): the
 * data is on its way, not broken, so the interface waits calmly instead of
 * showing an error.
 */
const notReadyCodes = new Set(["no_data", "not_available", "no_wind_field"])

/** Whether an error means "not collected yet" rather than a failure. */
export function isNotReady(error: unknown): boolean {
  return error instanceof ApiError && notReadyCodes.has(error.code)
}

/** Whether retrying a failed request could help: not for client errors or missing data. */
export function isRetryable(error: unknown): boolean {
  if (error instanceof DOMException && error.name === "AbortError") return false
  if (!(error instanceof ApiError)) return true
  return error.status >= 500 && !isNotReady(error)
}

async function errorFrom(response: Response): Promise<ApiError> {
  const body = (await response.json().catch(() => ({}))) as Envelope<unknown>
  return new ApiError(
    response.status,
    body.error?.code ?? "unexpected_response",
    body.error?.message ?? `Request failed with status ${response.status}`,
  )
}

/** GET a path on the API and unwrap the response envelope. */
export async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(path, { signal, headers: { Accept: "application/json" } })
  const body = (await response.json().catch(() => ({}))) as Envelope<T>
  if (!response.ok || body.error || body.data === undefined) {
    throw new ApiError(
      response.status,
      body.error?.code ?? "unexpected_response",
      body.error?.message ?? `Request failed with status ${response.status}`,
    )
  }
  return body.data
}

/**
 * GET map data that the API returns bare (GeoJSON, the weather grid, the
 * wind field). Errors are still wrapped in the error envelope.
 */
export async function apiGetBare<T>(
  path: string,
  signal?: AbortSignal,
  accept = "application/json",
): Promise<T> {
  const response = await fetch(path, { signal, headers: { Accept: accept } })
  if (!response.ok) throw await errorFrom(response)
  return (await response.json()) as T
}
