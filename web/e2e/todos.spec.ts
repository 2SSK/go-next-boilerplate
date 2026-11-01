import { expect, test } from "@playwright/test";
import { signIn, signOut } from "./auth";

test("todos need a signed-in user and return there after sign-in", async ({ page }) => {
  await page.goto("/todos");
  await expect(page).toHaveURL("/login?returnTo=%2Ftodos");

  await page.getByLabel("Username or email").fill("alice");
  await page.getByLabel("Password").fill("alice");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL("/todos");
  await expect(page.getByLabel("New todo")).toBeVisible();
});

test("a todo makes the full round trip and survives a reload", async ({ page }) => {
  const title = `e2e ${Date.now()}`;
  await signIn(page, "alice", "/todos");

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

test("each user sees only their own todos", async ({ page }) => {
  const title = `alice only ${Date.now()}`;
  await signIn(page, "alice", "/todos");
  await page.getByLabel("New todo").fill(title);
  await page.getByRole("button", { name: "Add" }).click();
  await expect(page.getByRole("listitem").filter({ hasText: title })).toBeVisible();
  await signOut(page);

  await signIn(page, "admin", "/todos");
  await expect(page.getByLabel("New todo")).toBeVisible();
  await expect(page.getByRole("listitem").filter({ hasText: title })).toHaveCount(0);
  await signOut(page);

  // Tidy up as alice
  await signIn(page, "alice", "/todos");
  await page
    .getByRole("listitem")
    .filter({ hasText: title })
    .getByRole("button", { name: `Delete "${title}"` })
    .click();
  await expect(page.getByRole("listitem").filter({ hasText: title })).toHaveCount(0);
});

test("navigation stays client-side and the tab title never visibly blanks", async ({ page }) => {
  await signIn(page, "alice");
  // A full reload would wipe this marker
  await page.evaluate(() => ((window as unknown as { marker: string }).marker = "same-document"));
  // Track the longest stretch with no <title>. A page that reads the session (like
  // /todos) gets its title with its per-request part, a frame or two after the old
  // one goes; static pages switch titles atomically. What must never come back is
  // the long gap (over a second) where the tab showed the URL.
  await page.evaluate(() => {
    const w = window as unknown as { longestBlankMs: number };
    w.longestBlankMs = 0;
    let blankSince: number | undefined;
    const sample = (now: number) => {
      if (!document.title) {
        blankSince ??= now;
        w.longestBlankMs = Math.max(w.longestBlankMs, now - blankSince);
      } else {
        blankSince = undefined;
      }
      requestAnimationFrame(sample);
    };
    requestAnimationFrame(sample);
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
    await page.evaluate(() => (window as unknown as { longestBlankMs: number }).longestBlankMs),
  ).toBeLessThan(100);
});
