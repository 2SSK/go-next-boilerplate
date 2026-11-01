"use client";

import { ThemeProvider as NextThemesProvider } from "next-themes";
import type { ComponentProps } from "react";

/** Class-based theming (".dark" on <html>), following the system until the user picks. */
export function ThemeProvider(props: Readonly<ComponentProps<typeof NextThemesProvider>>) {
  return (
    <NextThemesProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
      {...props}
    />
  );
}
