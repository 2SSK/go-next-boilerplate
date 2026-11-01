"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navigation, type NavbarStyle } from "@/config/site";
import { cn } from "@/lib/utils";

function isActive(pathname: string, href: string): boolean {
  return href === "/" ? pathname === "/" : pathname.startsWith(href);
}

const LINK_STYLES: Record<NavbarStyle, string> = {
  island: "rounded-full px-3.5",
  bar: "rounded-md px-3",
};

/**
 * The only client part of the navbar: it reads the URL to highlight the current link.
 * Everything around it renders on the server once and stays mounted across navigations.
 */
export function NavLinks({ variant }: Readonly<{ variant: NavbarStyle }>) {
  const pathname = usePathname();

  return navigation.map((item) => {
    const active = isActive(pathname, item.href);
    return (
      <Link
        key={item.href}
        href={item.href}
        aria-current={active ? "page" : undefined}
        className={cn(
          "interactive py-1.5 text-sm font-medium",
          LINK_STYLES[variant],
          active
            ? "bg-foreground/[0.07] text-foreground"
            : "text-muted-foreground hover:text-foreground",
        )}
      >
        {item.name}
      </Link>
    );
  });
}
