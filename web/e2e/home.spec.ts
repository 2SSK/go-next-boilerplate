import { expect, test } from "@playwright/test";

test("the landing page reaches the API and its database", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveTitle(/\S/);
  // The pill polls /readyz through Next.js: green means Go and Postgres both answered
  await expect(page.getByRole("status")).toContainText("API online", { timeout: 15_000 });
});

test("readiness is forwarded through Next.js", async ({ request }) => {
  const ready = await request.get("/readyz");
  expect(await ready.json()).toEqual({ status: "ok", database: "up" });
});

test("the API serves its own contract and docs", async ({ request }) => {
  const apiURL = process.env.E2E_API_URL ?? "http://localhost:8080";

  const spec = await request.get(`${apiURL}/openapi.yaml`);
  expect(spec.ok()).toBe(true);
  expect(await spec.text()).toContain("openapi: 3.");

  const docs = await request.get(`${apiURL}/docs`);
  expect(docs.ok()).toBe(true);
  expect(await docs.text()).toContain("createApiReference");
});
