import { NextResponse, type NextRequest } from "next/server";
import { authConfig, backchannelIssuer } from "@/lib/auth/config";
import { SESSION_COOKIE, sameOrigin, type Session, unseal } from "@/lib/auth/session";

/**
 * Signs out: ends the Keycloak session from the server (so no confirmation page),
 * then clears the cookie. POST only, from this site only.
 */
export async function POST(req: NextRequest) {
  if (!sameOrigin(req)) return new NextResponse("forbidden", { status: 403 });

  const session = await unseal<Session>(req.cookies.get(SESSION_COOKIE)?.value);
  if (session?.refreshToken) {
    await fetch(`${backchannelIssuer()}/protocol/openid-connect/logout`, {
      method: "POST",
      body: new URLSearchParams({
        client_id: authConfig.clientId,
        client_secret: authConfig.clientSecret,
        refresh_token: session.refreshToken,
      }),
    }).catch((err: unknown) => console.error("keycloak logout failed", err));
  }

  // 303: the browser follows a POST with a GET
  const res = NextResponse.redirect(new URL("/", authConfig.appUrl), 303);
  res.cookies.delete({ name: SESSION_COOKIE, path: "/" });
  return res;
}
