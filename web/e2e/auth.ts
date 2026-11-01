import { expect, type Page } from "@playwright/test";

/** Signs in through the app's own form and waits to land on `returnTo`. */
export async function signIn(page: Page, user: string, returnTo = "/"): Promise<void> {
  await page.goto(`/login?returnTo=${encodeURIComponent(returnTo)}`);
  await page.getByLabel("Username or email").fill(user);
  await page.getByLabel("Password").fill(user); // dev realm: password = username
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(returnTo);
}

export async function signOut(page: Page): Promise<void> {
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("link", { name: "Sign in" })).toBeVisible();
}
