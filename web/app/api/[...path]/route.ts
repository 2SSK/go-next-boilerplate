import { type NextRequest } from "next/server";
import { API_URL } from "@/lib/api-url";
import { getSession, sameOrigin } from "@/lib/auth/session";

/**
 * The browser's door to the Go API. It forwards the request and attaches the
 * session's access token on the server, so the token never reaches the browser.
 * Writes must come from this site's own pages: a CSRF defence on top of the
 * session cookie's SameSite=Lax.
 */
async function forward(req: NextRequest): Promise<Response> {
  const write = req.method !== "GET" && req.method !== "HEAD";
  if (write && !sameOrigin(req)) {
    return Response.json(
      { error: { code: "forbidden", message: "cross-site request refused" } },
      { status: 403 },
    );
  }

  const headers = new Headers();
  for (const name of ["content-type", "accept", "x-request-id"]) {
    const value = req.headers.get(name);
    if (value) headers.set(name, value);
  }
  const session = await getSession();
  if (session) headers.set("authorization", `Bearer ${session.accessToken}`);

  const upstream = await fetch(new URL(req.nextUrl.pathname + req.nextUrl.search, API_URL), {
    method: req.method,
    headers,
    body: write ? await req.arrayBuffer() : undefined,
    cache: "no-store",
    redirect: "manual",
  });

  const out = new Headers();
  for (const name of ["content-type", "x-request-id", "www-authenticate"]) {
    const value = upstream.headers.get(name);
    if (value) out.set(name, value);
  }
  return new Response(upstream.body, { status: upstream.status, headers: out });
}

export { forward as GET, forward as POST, forward as PATCH, forward as PUT, forward as DELETE };
