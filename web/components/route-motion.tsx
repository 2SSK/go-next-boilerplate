"use client";

import { usePathname } from "next/navigation";
import { useLayoutEffect, useState } from "react";

/**
 * Marks <html data-navigated> after the first client-side navigation. globals.css then
 * swaps the staggered `.animate-enter` entrance (fresh load only) for a single rise of the
 * whole page (`[data-navigated] main`), so moving around the app is quick and calm.
 */
export function RouteMotion() {
  const pathname = usePathname();
  const [initialPath] = useState(pathname);

  // Layout effect: the attribute must land before the new page's first paint
  useLayoutEffect(() => {
    if (pathname !== initialPath) document.documentElement.dataset.navigated = "";
  }, [pathname, initialPath]);

  return null;
}
