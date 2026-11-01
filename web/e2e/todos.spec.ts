import { expect, test } from "@playwright/test";

test("a todo makes the full round trip and survives a reload", async ({ page }) => {
  const title = `e2e ${Date.now()}`;
  await page.goto("/todos");

  await page.getByLabel("New todo").fill(title);
  await page.getByRole("button", { name: "Add" }).click();
  const row = page.getByRole("listitem").filter({ hasText: title });
  await expect(row).toBeVisible();

  // The checkbox flips at once (optimistic update); wait for the server to confirm
  // the save before reloading, or the reload can cancel the request mid-flight.
  const saved = page.waitForResponse(
    (r) => r.request().method() === "PATCH" && r.url().includes("/api/v1/todos/") && r.ok(),
  );
  await row.getByRole("checkbox").click();
  await expect(row.getByRole("checkbox")).toBeChecked();
  await saved;

  // Server-rendered from Postgres on reload: proves it was stored, not just shown
  await page.reload();
  const stored = page.getByRole("listitem").filter({ hasText: title });
  await expect(stored.getByRole("checkbox")).toBeChecked();

  await stored.getByRole("button", { name: `Delete "${title}"` }).click();
  await expect(page.getByRole("listitem").filter({ hasText: title })).toHaveCount(0);
});

test("navigation stays client-side and the tab title is never blank", async ({ page }) => {
  await page.goto("/");
  // A full reload would wipe this marker
  await page.evaluate(() => ((window as unknown as { marker: string }).marker = "same-document"));
  await page.evaluate(() => {
    const w = window as unknown as { blankTitleFrames: number };
    w.blankTitleFrames = 0;
    const sample = () => {
      if (!document.title) w.blankTitleFrames++;
      requestAnimationFrame(sample);
    };
    sample();
  });

  const nav = page.getByRole("navigation", { name: "Main" });
  for (const [link, path] of [
    ["Todos", "/todos"],
    ["Home", "/"],
    ["Todos", "/todos"],
    ["Home", "/"],
  ]) {
    await nav.getByRole("link", { name: link }).click();
    await expect(page).toHaveURL(path);
  }

  expect(await page.evaluate(() => (window as unknown as { marker?: string }).marker)).toBe(
    "same-document",
  );
  expect(
    await page.evaluate(() => (window as unknown as { blankTitleFrames: number }).blankTitleFrames),
  ).toBe(0);
});
