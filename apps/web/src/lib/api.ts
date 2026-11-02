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
