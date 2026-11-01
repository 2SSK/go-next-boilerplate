"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import * as client from "openid-client";
import { oidc } from "@/lib/auth/oidc";
import {
  SESSION_COOKIE,
  safeReturnTo,
  sealSession,
  sessionCookie,
  sessionFrom,
} from "@/lib/auth/session";

export interface SignInState {
  error?: string;
  /** Echoed back so a failed attempt doesn't clear what was typed. */
  username?: string;
}

/**
 * Signs in with the app's own form: the Next.js server exchanges the username and
 * password with Keycloak for tokens and keeps them in the encrypted session cookie.
 * The browser never sees a token. Next.js checks the Origin of every Server Action,
 * and Keycloak locks an account after repeated failures.
 */
export async function signIn(_prev: SignInState, form: FormData): Promise<SignInState> {
  const username = String(form.get("username") ?? "").trim();
  const password = String(form.get("password") ?? "");
  const returnTo = safeReturnTo(String(form.get("returnTo") ?? "/"));
  if (!username || !password) {
    return { error: "Enter your username and password.", username };
  }

  let tokens;
  try {
    tokens = await client.genericGrantRequest(await oidc(), "password", {
      username,
      password,
      scope: "openid profile email",
    });
  } catch (err) {
    if (err instanceof client.ResponseBodyError && err.error === "invalid_grant") {
      // One message for every credential problem, so it reveals nothing.
      return { error: "Incorrect username or password.", username };
    }
    console.error("sign-in failed", err);
    return { error: "Sign-in is unavailable right now. Try again in a moment.", username };
  }

  (await cookies()).set(SESSION_COOKIE, await sealSession(sessionFrom(tokens)), sessionCookie);
  redirect(returnTo);
}
