"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { navigation } from "@/config/site";
import { ThemeToggle } from "./theme-toggle";

function isActive(pathname: string, href: string): boolean {
  return href === "/" ? pathname === "/" : pathname.startsWith(href);
}

/** A frosted capsule floating at the top while the page scrolls under it. */
export function Navbar() {
  const pathname = usePathname();

  return (
    <header className="pointer-events-none fixed inset-x-0 top-3 z-50 flex justify-center px-4 sm:top-4">
      <nav
        aria-label="Main"
        className="bg-card/75 shadow-float pointer-events-auto flex items-center gap-1 rounded-full border p-1 backdrop-blur-xl backdrop-saturate-150"
      >
        {navigation.map((item) => {
          const active = isActive(pathname, item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "interactive rounded-full px-3.5 py-1.5 text-sm font-medium",
                active
                  ? "bg-foreground/[0.07] text-foreground"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {item.name}
            </Link>
          );
        })}
        <span className="bg-border mx-0.5 h-4 w-px" aria-hidden="true" />
        <ThemeToggle />
      </nav>
    </header>
  );
}
