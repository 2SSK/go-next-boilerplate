"use client";

import { Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();

  const toggle = (): void => {
    const next = resolvedTheme === "dark" ? "light" : "dark";
    // Apply the theme to <html> synchronously: next-themes would do it in an effect,
    // after the view transition has already taken its "after" snapshot.
    const apply = (): void => {
      const root = document.documentElement;
      root.classList.remove("light", "dark");
      root.classList.add(next);
      root.style.colorScheme = next;
      setTheme(next);
    };

    if (!document.startViewTransition) {
      apply();
      return;
    }
    // A short cross-fade (globals.css); the attribute scopes it to theme switches.
    const root = document.documentElement;
    root.dataset.themeSwitch = "";
    document.startViewTransition(apply).finished.finally(() => {
      delete root.dataset.themeSwitch;
    });
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
