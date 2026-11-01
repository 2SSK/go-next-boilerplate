/**
 * Fetch helper for the Go API. Requests go to same-origin paths ("/api/...",
 * "/healthz") and next.config.ts rewrites them to API_URL, so the browser never
 * needs CORS and the API address lives in one env var.
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

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });

  if (!res.ok) {
    // The API answers errors as {"error": "..."}; fall back to the status text.
    const body: unknown = await res.json().catch(() => null);
    const message =
      body !== null && typeof body === "object" && "error" in body && typeof body.error === "string"
        ? body.error
        : res.statusText;
    throw new ApiError(message, res.status);
  }

  return (res.status === 204 ? undefined : await res.json()) as T;
}
