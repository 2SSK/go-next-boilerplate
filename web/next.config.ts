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
  // The browser only talks to Next.js; these forward to the Go API.
  async rewrites() {
    return [
      { source: "/api/:path*", destination: `${API_URL}/api/:path*` },
      { source: "/healthz", destination: `${API_URL}/healthz` },
    ];
  },
};

export default nextConfig;
