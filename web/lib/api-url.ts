/**
 * Where the Go API listens, as seen from the Next.js server. The browser never uses it:
 * app/api/[...path]/route.ts and next.config.ts forward here, and server components fetch it directly.
 */
export const API_URL = process.env.API_URL ?? "http://localhost:8080";
