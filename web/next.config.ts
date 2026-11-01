import type { NextConfig } from "next";

/** Where the Go API listens. The browser only ever talks to Next, which proxies. */
const API_URL = process.env.API_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  async rewrites() {
    return [
      { source: "/api/:path*", destination: `${API_URL}/api/:path*` },
      { source: "/healthz", destination: `${API_URL}/healthz` },
    ];
  },
};

export default nextConfig;
