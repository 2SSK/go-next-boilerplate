import { defineConfig, devices } from "@playwright/test";

/**
 * End-to-end tests against a running stack: browser → Next.js → Go → Postgres.
 * Start it first (`make up`), then `make e2e`. E2E_BASE_URL (the web app) and
 * E2E_API_URL (the API, for its docs) point elsewhere.
 */
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
