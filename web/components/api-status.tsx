"use client";

import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";

type Status = { state: "checking" } | { state: "up"; ms: number } | { state: "down" };

const POLL_MS = 10_000;

/**
 * A live pill showing whether the Go API (and its database) answers /healthz. It polls
 * only while visible: a hidden tab, or a route Next.js keeps hidden after navigating away
 * (effects are cleaned up then), sends no requests.
 */
export function ApiStatus() {
  const [status, setStatus] = useState<Status>({ state: "checking" });

  useEffect(() => {
    let cancelled = false;

    const check = async (): Promise<void> => {
      const start = performance.now();
      let next: Status;
      try {
        const res = await fetch("/healthz", { cache: "no-store" });
        next = res.ok
          ? { state: "up", ms: Math.round(performance.now() - start) }
          : { state: "down" };
      } catch {
        next = { state: "down" };
      }
      if (!cancelled) setStatus(next);
    };

    // Poll only while the tab is visible, and check at once when it becomes visible again
    const tick = (): void => {
      if (document.visibilityState === "visible") void check();
    };
    tick();
    const id = setInterval(tick, POLL_MS);
    document.addEventListener("visibilitychange", tick);
    return () => {
      cancelled = true;
      clearInterval(id);
      document.removeEventListener("visibilitychange", tick);
    };
  }, []);

  const label = {
    checking: "Checking API",
    up: "API online",
    down: "API offline",
  }[status.state];

  return (
    <span
      role="status"
      className="bg-card text-muted-foreground shadow-card inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium"
    >
      <span className="relative flex size-2">
        {status.state === "up" && (
          <span className="bg-success absolute inline-flex size-full animate-ping rounded-full opacity-60" />
        )}
        <span
          className={cn(
            "relative inline-flex size-2 rounded-full",
            status.state === "up" && "bg-success",
            status.state === "down" && "bg-destructive",
            status.state === "checking" && "bg-subtle",
          )}
        />
      </span>
      {label}
      {status.state === "up" && <span className="meta text-[11px]">{status.ms}ms</span>}
    </span>
  );
}
