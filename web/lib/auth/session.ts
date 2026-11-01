import "server-only";
import { decodeJwt, EncryptJWT, jwtDecrypt } from "jose";
import { cookies } from "next/headers";
import type { TokenEndpointResponse, TokenEndpointResponseHelpers } from "openid-client";
import { authConfig, authSecret } from "./config";

export const SESSION_COOKIE = "app_session";

/** What the session cookie holds: just the tokens, to stay under 4 KB. */
export interface Session {
  accessToken: string;
  refreshToken?: string;
  /** When the access token expires, in Unix seconds. */
  expiresAt: number;
}

/** The signed-in user, read from the access token. */
export interface User {
  id: string;
  name: string;
  email?: string;
  roles: string[];
}

const secure = authConfig.appUrl.startsWith("https://");

export const sessionCookie = {
  httpOnly: true,
  sameSite: "lax" as const,
  secure,
  path: "/",
  // Matches Keycloak's SSO idle timeout; refresh keeps it alive while in use.
  maxAge: 60 * 60 * 8,
};

let keyPromise: Promise<Uint8Array> | undefined;
function key(): Promise<Uint8Array> {
  keyPromise ??= crypto.subtle
    .digest("SHA-256", new TextEncoder().encode(authSecret()))
    .then((d) => new Uint8Array(d));
  return keyPromise;
}

/** Encrypts payload into a cookie value (JWE, A256GCM) that expires after ttl. */
export async function seal(payload: object, ttl: string): Promise<string> {
  return new EncryptJWT({ ...payload })
    .setProtectedHeader({ alg: "dir", enc: "A256GCM" })
    .setIssuedAt()
    .setExpirationTime(ttl)
    .encrypt(await key());
}

/** Decrypts a cookie value from seal; undefined if missing, tampered, or expired. */
export async function unseal<T>(value: string | undefined): Promise<T | undefined> {
  if (!value) return undefined;
  try {
    const { payload } = await jwtDecrypt(value, await key());
    return payload as T;
  } catch {
    return undefined;
  }
}

export function sessionFrom(tokens: TokenEndpointResponse & TokenEndpointResponseHelpers): Session {
  return {
    accessToken: tokens.access_token,
    refreshToken: tokens.refresh_token,
    expiresAt: Math.floor(Date.now() / 1000) + (tokens.expiresIn() ?? 300),
  };
}

export function sealSession(session: Session): Promise<string> {
  return seal(session, `${sessionCookie.maxAge}s`);
}

/** The current request's session, if signed in. */
export async function getSession(): Promise<Session | undefined> {
  return unseal<Session>((await cookies()).get(SESSION_COOKIE)?.value);
}

/** Display details from the access token. The API verifies it; this only reads it. */
export function userFrom(session: Session): User {
  const claims = decodeJwt(session.accessToken) as {
    sub?: string;
    name?: string;
    preferred_username?: string;
    email?: string;
    realm_access?: { roles?: string[] };
  };
  return {
    id: claims.sub ?? "",
    name: claims.name ?? claims.preferred_username ?? "Signed in",
    email: claims.email,
    roles: claims.realm_access?.roles ?? [],
  };
}

/** Only relative paths on this site, so login can't become an open redirect. */
export function safeReturnTo(value: string | null | undefined): string {
  return value && value.startsWith("/") && !value.startsWith("//") ? value : "/";
}

/** True when the request comes from this site's own pages (CSRF defence for writes). */
export function sameOrigin(req: Request): boolean {
  const origin = req.headers.get("origin");
  if (origin) return origin === new URL(authConfig.appUrl).origin;
  return req.headers.get("sec-fetch-site") === "same-origin";
}
