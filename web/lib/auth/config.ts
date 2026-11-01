import "server-only";

/**
 * Login settings. The web app's Next.js server is the OAuth client: it runs the
 * login, holds the tokens in an encrypted cookie, and attaches them to API calls.
 * Browsers never see a token.
 */
export const authConfig = {
  /** Public URL of this app, for the OAuth redirect (APP_URL). */
  appUrl: process.env.APP_URL ?? "http://localhost:3000",
  /** The issuer browsers and tokens use (AUTH_ISSUER). */
  issuer: process.env.AUTH_ISSUER ?? "http://localhost:8180/realms/app",
  /** Where this server reaches the issuer, if different (AUTH_INTERNAL_ISSUER; Docker). */
  internalIssuer: process.env.AUTH_INTERNAL_ISSUER || undefined,
  clientId: process.env.AUTH_CLIENT_ID ?? "web",
  clientSecret: process.env.AUTH_CLIENT_SECRET ?? "dev-web-secret",
};

/** The secret that encrypts session cookies. Required in production. */
export function authSecret(): string {
  const secret = process.env.AUTH_SECRET;
  if (secret && secret.length >= 32) return secret;
  if (process.env.NODE_ENV === "production" && !process.env.AUTH_ALLOW_DEV_SECRET) {
    throw new Error("AUTH_SECRET must be set to at least 32 random characters in production");
  }
  return "development-only-secret-do-not-use-in-production";
}

/** Where this server talks to the provider: the internal address if set. */
export function backchannelIssuer(): string {
  return authConfig.internalIssuer ?? authConfig.issuer;
}
