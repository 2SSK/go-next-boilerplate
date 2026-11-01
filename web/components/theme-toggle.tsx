"use client";

import { Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();

  const toggle = (): void => {
    const next = resolvedTheme === "dark" ? "light" : "dark";
    // A short cross-fade (globals.css) where supported, an instant switch elsewhere
    if (document.startViewTransition) {
      document.startViewTransition(() => setTheme(next));
    } else {
      setTheme(next);
    }
  };

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label="Toggle theme"
      title="Toggle theme"
      className="interactive text-muted-foreground hover:bg-foreground/[0.06] hover:text-foreground flex size-8 items-center justify-center rounded-full active:scale-95"
    >
      {/* Both icons render; CSS picks one so the server render never mismatches */}
      <Sun className="hidden size-4 dark:block" />
      <Moon className="block size-4 dark:hidden" />
    </button>
  );
}
