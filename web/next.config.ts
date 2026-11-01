import type { NextConfig } from "next";
import { API_URL } from "./lib/api-url";

const isDev = process.env.NODE_ENV === "development";

/**
 * One static Content-Security-Policy for every page. Nonces would be stricter, but
 * they force every page to render per request and would discard the prebuilt static
 * shells (see node_modules/next/dist/docs/01-app/02-guides/content-security-policy.md).
 * The cost: inline scripts are allowed (Next's bootstrap and the theme script need
 * them), while scripts from any other origin are not. upgrade-insecure-requests is
 * left out so `make up` works over plain HTTP; HTTPS and HSTS belong to whatever
 * terminates TLS in production.
 */
const contentSecurityPolicy = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ""}`,
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' blob: data:",
  "font-src 'self'",
  `connect-src 'self'${isDev ? " ws: wss:" : ""}`,
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
].join("; ");

const securityHeaders = [
  { key: "Content-Security-Policy", value: contentSecurityPolicy },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
];

const nextConfig: NextConfig = {
  poweredByHeader: false,
  output: "standalone",
  // Static shell + streamed dynamic parts (Partial Prerendering), and prefetch of each
  // route's shell so client navigations render without waiting on the server.
  cacheComponents: true,
  partialPrefetching: true,
  // Memoizes components and hooks at build time; no hand-written useMemo/useCallback.
  reactCompiler: true,
  // The browser only talks to Next.js for the app; these forward to the Go API.
  // The API's docs (/docs, /openapi.yaml) are served by the API itself, not proxied.
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
  async rewrites() {
    return ["/api/:path*", "/readyz", "/livez"].map((source) => ({
      source,
      destination: `${API_URL}${source}`,
    }));
  },
};

export default nextConfig;
