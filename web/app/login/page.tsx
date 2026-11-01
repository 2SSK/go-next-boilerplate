import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { LoginForm } from "@/components/login-form";
import { getSession, safeReturnTo } from "@/lib/auth/session";

export const metadata: Metadata = { title: "Sign in" };

/** The heading is static; the form depends on the request (cookie, returnTo). */
export default function LoginPage(props: PageProps<"/login">) {
  return (
    <main className="container-page page-top">
      <div className="mx-auto max-w-sm">
        <h1 className="animate-enter text-[32px] tracking-[-0.02em]">Sign in</h1>
        <p className="animate-enter text-muted-foreground mt-2" style={{ animationDelay: "60ms" }}>
          Use your account to continue.
        </p>
        <div className="animate-enter mt-8" style={{ animationDelay: "120ms" }}>
          <Suspense fallback={<div className="surface h-[300px]" aria-hidden="true" />}>
            <Form searchParams={props.searchParams} />
          </Suspense>
        </div>
      </div>
    </main>
  );
}

async function Form({ searchParams }: Pick<PageProps<"/login">, "searchParams">) {
  const params = await searchParams;
  const returnTo = safeReturnTo(typeof params.returnTo === "string" ? params.returnTo : null);
  // Already signed in: go where the visitor was headed.
  if (await getSession()) redirect(returnTo);
  return <LoginForm returnTo={returnTo} devHint={process.env.NODE_ENV !== "production"} />;
}
