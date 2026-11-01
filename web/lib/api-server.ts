import "server-only";
import { API_URL } from "./api-url";
import { parseResponse } from "./api";

/**
 * Calls the API from a server component, straight to API_URL with no browser round trip.
 * The result is uncached, so render the caller inside <Suspense>: the page's static shell
 * ships at once and this data streams in behind the fallback.
 */
export async function serverApi<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  return parseResponse<T>(res);
}
