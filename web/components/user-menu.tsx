import { LogIn, LogOut } from "lucide-react";
import { getSession, userFrom } from "@/lib/auth/session";

/**
 * Who is signed in, with sign in / sign out. It reads the session cookie, so it
 * renders per request; the navbar wraps it in <Suspense> to keep the rest of the
 * page in the static shell.
 */
export async function UserMenu() {
  const session = await getSession();
  if (!session) {
    return (
      <a
        href="/login"
        className="interactive text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium"
      >
        <LogIn className="size-4" />
        Sign in
      </a>
    );
  }
  const user = userFrom(session);
  return (
    <form action="/auth/logout" method="post" className="flex items-center gap-1">
      <span className="max-w-32 truncate px-2 text-sm font-medium" title={user.email ?? user.name}>
        {user.name}
      </span>
      <button
        type="submit"
        aria-label="Sign out"
        title="Sign out"
        className="interactive text-muted-foreground hover:bg-foreground/[0.06] hover:text-foreground flex size-8 items-center justify-center rounded-full active:scale-95"
      >
        <LogOut className="size-4" />
      </button>
    </form>
  );
}

/** Same footprint as the menu, so nothing moves when it streams in. */
export function UserMenuPlaceholder() {
  return <span className="h-8 w-20" aria-hidden="true" />;
}
