import { NextResponse, type NextRequest } from "next/server";
import * as client from "openid-client";
import { protectedPaths } from "@/config/site";
import { oidc } from "@/lib/auth/oidc";
import {
  SESSION_COOKIE,
  type Session,
  sealSession,
  sessionCookie,
  sessionFrom,
  unseal,
} from "@/lib/auth/session";

/** Refresh this long before the access token actually expires. */
const REFRESH_MARGIN_SECONDS = 60;

/**
 * Runs before every page and API request:
 * 1. refreshes an access token that is about to expire, and hands the new cookie
 *    both to the browser and to the page rendering this request;
 * 2. sends signed-out visitors of protected pages to the login.
 */
export async function proxy(req: NextRequest) {
  let session = await unseal<Session>(req.cookies.get(SESSION_COOKIE)?.value);
  let cookie: string | null | undefined; // string: new value, null: clear, undefined: unchanged

  if (session && session.expiresAt - Date.now() / 1000 < REFRESH_MARGIN_SECONDS) {
    try {
      if (!session.refreshToken) throw new Error("no refresh token");
      const tokens = await client.refreshTokenGrant(await oidc(), session.refreshToken);
      session = sessionFrom(tokens);
      cookie = await sealSession(session);
    } catch {
      // Refresh token expired or revoked: the session is over.
      session = undefined;
      cookie = null;
    }
  }

  const path = req.nextUrl.pathname;
  if (!session && protectedPaths.some((p) => path === p || path.startsWith(`${p}/`))) {
    const login = new URL("/login", req.url);
    login.searchParams.set("returnTo", path + req.nextUrl.search);
    const res = NextResponse.redirect(login);
    if (cookie === null) res.cookies.delete(SESSION_COOKIE);
    return res;
  }

  if (cookie === undefined) return NextResponse.next();

  // Pass the new cookie upstream so this very request renders with it.
  if (cookie) req.cookies.set(SESSION_COOKIE, cookie);
  else req.cookies.delete(SESSION_COOKIE);
  const res = NextResponse.next({ request: { headers: req.headers } });
  if (cookie) res.cookies.set(SESSION_COOKIE, cookie, sessionCookie);
  else res.cookies.delete(SESSION_COOKIE);
  return res;
}

export const config = {
  // Everything except static assets and the API's own health probes.
  matcher: ["/((?!_next/static|_next/image|favicon.ico|icon.svg|readyz|livez).*)"],
};
