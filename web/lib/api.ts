/**
 * Typed client for the Go API, generated from api/openapi.yaml (`make generate`).
 *
 * In the browser, `api` calls same-origin paths ("/api/...", "/readyz") and
 * next.config.ts forwards them to API_URL, so there is no CORS and no API host in
 * client code. Server components use `serverApi` from lib/api-server.ts instead.
 */
import createClient from "openapi-fetch";
import type { components, paths } from "./api/schema";

export type ErrorBody = components["schemas"]["Error"];
export type ErrorCode = ErrorBody["error"]["code"];

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    /** The contract's machine-readable code; branch on this, not on the message. */
    readonly code?: ErrorCode,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

function isErrorBody(v: unknown): v is ErrorBody {
  return typeof v === "object" && v !== null && "error" in v && typeof v.error === "object";
}

/** Returns the response data, or throws ApiError with the API's own message and code. */
export async function unwrap<D>(
  request: Promise<{ data?: D; error?: unknown; response: Response }>,
): Promise<D> {
  const { data, error, response } = await request;
  if (!response.ok) {
    if (isErrorBody(error))
      throw new ApiError(error.error.message, response.status, error.error.code);
    throw new ApiError(response.statusText, response.status);
  }
  return data as D;
}

/** A message fit to show people: the API's own message, or a hint that it is down. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  return "Could not reach the API. Is it running?";
}

/** Calls the API from the browser, through the same-origin rewrites. */
export const api = createClient<paths>();
