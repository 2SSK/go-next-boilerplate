import "server-only";
import createClient from "openapi-fetch";
import type { paths } from "./api/schema";
import { API_URL } from "./api-url";

/**
 * Calls the API from a server component, straight to API_URL with no browser round
 * trip. Results are uncached, so render the caller inside <Suspense>: the page's
 * static shell ships at once and this data streams in behind the fallback.
 */
export const serverApi = createClient<paths>({ baseUrl: API_URL });
