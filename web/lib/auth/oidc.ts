import "server-only";
import * as client from "openid-client";
import { authConfig, backchannelIssuer } from "./config";

let configuration: Promise<client.Configuration> | undefined;

/**
 * The OIDC client configuration, discovered once and cached. Discovery goes to the
 * backchannel address, but the issuer must be the public one, which is what tokens
 * carry. A failed discovery is retried on the next call.
 */
export function oidc(): Promise<client.Configuration> {
  configuration ??= discover().catch((err: unknown) => {
    configuration = undefined;
    throw err;
  });
  return configuration;
}

async function discover(): Promise<client.Configuration> {
  const base = backchannelIssuer();
  const res = await fetch(`${base}/.well-known/openid-configuration`, { cache: "no-store" });
  if (!res.ok) throw new Error(`OIDC discovery at ${base} failed: ${res.status}`);
  const metadata = (await res.json()) as client.ServerMetadata;
  if (metadata.issuer !== authConfig.issuer) {
    throw new Error(`issuer is ${metadata.issuer}, expected ${authConfig.issuer} (AUTH_ISSUER)`);
  }

  const config = new client.Configuration(metadata, authConfig.clientId, authConfig.clientSecret);
  // Plain HTTP is only acceptable for a provider on this machine or the Compose network.
  if (new URL(base).protocol === "http:" && isLocal(authConfig.issuer)) {
    client.allowInsecureRequests(config);
  }
  return config;
}

function isLocal(url: string): boolean {
  const { hostname } = new URL(url);
  return hostname === "localhost" || hostname === "127.0.0.1";
}
