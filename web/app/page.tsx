import Link from "next/link"; // feature:todo
import { ArrowRight } from "lucide-react"; // feature:todo
import { ApiStatus } from "@/components/api-status";
import { site } from "@/config/site";

/** Staggered entrance: each block rises in 60 ms after the one before. */
function rise(step: number): React.CSSProperties {
  return { animationDelay: `${step * 60}ms` };
}

const ENDPOINTS = [
  { method: "GET", path: "/", note: "API name and links" },
  { method: "GET", path: "/livez", note: "Liveness: the process is up" },
  { method: "GET", path: "/readyz", note: "Readiness: database included" },
  { method: "GET", path: "/docs", note: "Interactive docs, served by the API" },
  // feature:todo:begin
  { method: "GET", path: "/api/v1/todos", note: "List todos" },
  { method: "POST", path: "/api/v1/todos", note: "Create a todo" },
  { method: "PATCH", path: "/api/v1/todos/{id}", note: "Rename or complete" },
  { method: "DELETE", path: "/api/v1/todos/{id}", note: "Remove a todo" },
  // feature:todo:end
] as const;

const STACK = ["Go net/http", "pgx", "Postgres", "Next.js", "shadcn/ui", "Tailwind"] as const;

export default function Home() {
  return (
    <main className="container-page page-top">
      <div className="animate-enter" style={rise(0)}>
        <ApiStatus />
      </div>

      <h1
        className="animate-enter mt-7 text-[40px] leading-[1.08] tracking-[-0.03em] sm:text-[56px]"
        style={rise(1)}
      >
        {site.name}
      </h1>

      <p
        className="animate-enter text-muted-foreground mt-4 max-w-[52ch] text-[17px] leading-relaxed sm:text-[19px]"
        style={rise(2)}
      >
        {site.description}
      </p>

      <div className="animate-enter mt-8 flex flex-wrap items-center gap-2" style={rise(3)}>
        {/* feature:todo:begin */}
        <Link
          href="/todos"
          className="group interactive bg-primary text-primary-foreground inline-flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold hover:opacity-90 active:scale-[0.98]"
        >
          Open the todo demo
          <ArrowRight className="interactive size-4 group-hover:translate-x-0.5" />
        </Link>
        {/* feature:todo:end */}
        <a
          href={site.repo}
          className="interactive bg-card text-muted-foreground hover:text-foreground hover:shadow-card inline-flex items-center rounded-full border px-4 py-2 text-sm font-medium active:scale-[0.98]"
        >
          Read the README
        </a>
      </div>

      <section className="animate-enter mt-16" style={rise(4)}>
        <h2 className="label">Endpoints</h2>
        <ul className="surface mt-3 divide-y overflow-hidden">
          {ENDPOINTS.map((e) => (
            <li key={`${e.method} ${e.path}`} className="flex items-center gap-4 px-5 py-3">
              <span className="text-primary w-14 font-mono text-xs font-semibold">{e.method}</span>
              <code className="flex-1 truncate font-mono text-sm">{e.path}</code>
              <span className="text-muted-foreground hidden text-sm sm:block">{e.note}</span>
            </li>
          ))}
        </ul>
      </section>

      <section className="animate-enter mt-12" style={rise(5)}>
        <h2 className="label">Stack</h2>
        <ul className="mt-3 flex flex-wrap gap-1.5">
          {STACK.map((s) => (
            <li
              key={s}
              className="bg-foreground/[0.05] text-muted-foreground rounded-full px-2.5 py-0.5 font-mono text-xs font-medium"
            >
              {s}
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
