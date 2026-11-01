"use client";

import { useActionState } from "react";
import { Loader2 } from "lucide-react";
import { signIn, type SignInState } from "@/app/login/actions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const initial: SignInState = {};

export function LoginForm({ returnTo, devHint }: Readonly<{ returnTo: string; devHint: boolean }>) {
  const [state, action, pending] = useActionState(signIn, initial);

  return (
    <>
      <form action={action} className="surface space-y-4 p-6">
        <input type="hidden" name="returnTo" value={returnTo} />
        <div className="space-y-1.5">
          <label htmlFor="username" className="text-sm font-medium">
            Username or email
          </label>
          <Input
            id="username"
            name="username"
            autoComplete="username"
            defaultValue={state.username}
            required
            autoFocus
            className="h-10"
          />
        </div>
        <div className="space-y-1.5">
          <label htmlFor="password" className="text-sm font-medium">
            Password
          </label>
          <Input
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            required
            className="h-10"
          />
        </div>

        {state.error && (
          <p
            role="alert"
            className="bg-destructive/[0.06] text-destructive rounded-lg px-3 py-2 text-sm"
          >
            {state.error}
          </p>
        )}

        <Button type="submit" size="lg" disabled={pending} className="h-10 w-full rounded-full">
          {pending && <Loader2 className="animate-spin" />}
          Sign in
        </Button>
      </form>
      {devHint && (
        <p className="meta mt-4 text-center text-xs">Dev users: alice / alice · admin / admin</p>
      )}
    </>
  );
}
