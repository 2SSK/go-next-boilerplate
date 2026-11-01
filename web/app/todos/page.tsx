import type { Metadata } from "next";
import { TodoList } from "@/components/todos/todo-list";

export const metadata: Metadata = { title: "Todos" };

export default function TodosPage() {
  return (
    <main className="container-page pt-32 sm:pt-40">
      <p className="label animate-rise">Demo</p>
      <h1
        className="animate-rise mt-2 text-[32px] tracking-[-0.02em]"
        style={{ animationDelay: "60ms" }}
      >
        Todos
      </h1>
      <p className="animate-rise text-muted-foreground mt-2" style={{ animationDelay: "120ms" }}>
        A full round trip: Next.js → Go → Postgres. Run{" "}
        <code className="font-mono text-sm">make reset</code> to remove it.
      </p>
      <div className="animate-rise mt-8" style={{ animationDelay: "180ms" }}>
        <TodoList />
      </div>
    </main>
  );
}
