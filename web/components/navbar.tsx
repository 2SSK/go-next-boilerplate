import Link from "next/link";
import { Terminal } from "lucide-react";
import { site } from "@/config/site";
import { NavLinks } from "./nav-links";
import { ThemeToggle } from "./theme-toggle";

/**
 * The top navigation, in the style set by `site.navbar`. A server component rendered from
 * the root layout, so it is sent once and survives every client navigation; only
 * <NavLinks> and <ThemeToggle> hydrate.
 */
export function Navbar() {
  return site.navbar === "bar" ? <BarNav /> : <IslandNav />;
}

/** A frosted capsule floating at the top while the page scrolls under it. */
function IslandNav() {
  return (
    <header className="pointer-events-none fixed inset-x-0 top-3 z-50 flex justify-center px-4 sm:top-4">
      <nav
        aria-label="Main"
        className="bg-card/75 shadow-float pointer-events-auto flex items-center gap-1 rounded-full border p-1 backdrop-blur-xl backdrop-saturate-150"
      >
        <NavLinks variant="island" />
        <span className="bg-border mx-0.5 h-4 w-px" aria-hidden="true" />
        <ThemeToggle />
      </nav>
    </header>
  );
}

/** A full-width bar that sticks to the top: app name on the left, links on the right. */
function BarNav() {
  return (
    <header className="bg-background/80 sticky top-0 z-50 border-b backdrop-blur-xl backdrop-saturate-150">
      <div className="container-page flex h-14 items-center justify-between gap-4">
        <Link href="/" className="flex items-center gap-2.5 text-sm font-semibold tracking-tight">
          <span className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-md">
            <Terminal className="size-3.5" strokeWidth={2.5} />
          </span>
          {site.name}
        </Link>

        <nav aria-label="Main" className="flex items-center gap-1">
          <NavLinks variant="bar" />
          <span className="bg-border mx-1 h-4 w-px" aria-hidden="true" />
          <ThemeToggle />
        </nav>
      </div>
    </header>
  );
}
