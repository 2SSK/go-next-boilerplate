/**
 * Fetch helpers for the Go API, shared by the browser and the server.
 *
 * In the browser, `api()` calls same-origin paths ("/api/...", "/healthz") and
 * next.config.ts forwards them to API_URL, so there is no CORS and no API host in client
 * code. Server components use `serverApi()` from lib/api-server.ts instead.
 */

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** Returns the parsed body, or throws ApiError with the API's {"error": "..."} message. */
export async function parseResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body: unknown = await res.json().catch(() => null);
    const message =
      body !== null && typeof body === "object" && "error" in body && typeof body.error === "string"
        ? body.error
        : res.statusText;
    throw new ApiError(message, res.status);
  }
  return (res.status === 204 ? undefined : await res.json()) as T;
}

/** A message fit to show people: the API's own message, or a hint that it is down. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  return "Could not reach the API. Is it running?";
}

/** Calls the API from the browser. */
export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  return parseResponse<T>(res);
}
