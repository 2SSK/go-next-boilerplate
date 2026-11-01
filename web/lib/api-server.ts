import "server-only";
import createClient from "openapi-fetch";
import type { paths } from "./api/schema";
import { API_URL } from "./api-url";
import { getSession } from "./auth/session";

/**
 * A client for calling the API from a server component, straight to API_URL, with
 * the signed-in user's access token attached. Results are uncached, so render the
 * caller inside <Suspense>: the page's static shell ships at once and this data
 * streams in behind the fallback.
 */
export async function serverApi() {
  const session = await getSession();
  return createClient<paths>({
    baseUrl: API_URL,
    headers: session ? { Authorization: `Bearer ${session.accessToken}` } : {},
  });
}
