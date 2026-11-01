import { expect, test } from "@playwright/test";
import { signIn, signOut } from "./auth";

test("wrong credentials show an error and keep the username", async ({ page }) => {
  await page.goto("/login");
  // A user that doesn't exist: Keycloak's lockout only counts real accounts,
  // so repeated runs can never lock out a dev user.
  await page.getByLabel("Username or email").fill("no-such-user");
  await page.getByLabel("Password").fill("wrong");
  await page.getByRole("button", { name: "Sign in" }).click();

  // Scoped to <main>: Next.js also renders a hidden route announcer with role="alert"
  await expect(page.getByRole("main").getByRole("alert")).toHaveText(
    "Incorrect username or password.",
  );
  await expect(page.getByLabel("Username or email")).toHaveValue("no-such-user");
  await expect(page).toHaveURL(/\/login/);
});

test("signing in shows the user, signing out ends the session", async ({ page, context }) => {
  await signIn(page, "alice");
  await expect(page.getByRole("button", { name: "Sign out" })).toBeVisible();
  await expect(page.getByText("Alice Example")).toBeVisible();

  // The session never reaches page JavaScript
  expect(await page.evaluate(() => document.cookie)).not.toContain("app_session");
  const cookie = (await context.cookies()).find((c) => c.name === "app_session");
  expect(cookie?.httpOnly).toBe(true);

  await signOut(page);
  expect((await context.cookies()).find((c) => c.name === "app_session")).toBeUndefined();
});
