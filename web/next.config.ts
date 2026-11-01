import type { NextConfig } from "next";
import { API_URL } from "./lib/api-url";

const nextConfig: NextConfig = {
  output: "standalone",
  // Static shell + streamed dynamic parts (Partial Prerendering), and prefetch of each
  // route's shell so client navigations render without waiting on the server.
  cacheComponents: true,
  partialPrefetching: true,
  // Memoizes components and hooks at build time; no hand-written useMemo/useCallback.
  reactCompiler: true,
  // The browser only talks to Next.js for the app; these forward to the Go API.
  // The API's docs (/docs, /openapi.yaml) are served by the API itself, not proxied.
  async rewrites() {
    return ["/api/:path*", "/readyz", "/livez"].map((source) => ({
      source,
      destination: `${API_URL}${source}`,
    }));
  },
};

export default nextConfig;
