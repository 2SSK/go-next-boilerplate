import type { Metadata } from "next";
import { Suspense } from "react";
import { TodoList, TodoListSkeleton } from "@/components/todos/todo-list";
import { errorMessage } from "@/lib/api";
import { serverApi } from "@/lib/api-server";
import { type Todo, TODOS_PATH } from "@/lib/todos";

export const metadata: Metadata = { title: "Todos" };

/**
 * The heading is part of the static shell, served instantly. The list loads on the server
 * and streams in behind the skeleton, so there is no client-side fetch after hydration.
 */
export default function TodosPage() {
  return (
    <main className="container-page page-top">
      <p className="label animate-enter">Demo</p>
      <h1
        className="animate-enter mt-2 text-[32px] tracking-[-0.02em]"
        style={{ animationDelay: "60ms" }}
      >
        Todos
      </h1>
      <p className="animate-enter text-muted-foreground mt-2" style={{ animationDelay: "120ms" }}>
        A full round trip: Next.js → Go → Postgres. Run{" "}
        <code className="font-mono text-sm">make reset</code> to remove it.
      </p>
      <div className="animate-enter mt-8" style={{ animationDelay: "180ms" }}>
        <Suspense fallback={<TodoListSkeleton />}>
          <Todos />
        </Suspense>
      </div>
    </main>
  );
}

async function Todos() {
  const { todos, error } = await loadTodos();
  return <TodoList initialTodos={todos} initialError={error} />;
}

/** Fetch only; a failed load becomes an inline error, not a broken page. */
async function loadTodos(): Promise<{ todos: Todo[]; error?: string }> {
  try {
    return { todos: await serverApi<Todo[]>(TODOS_PATH) };
  } catch (err) {
    return { todos: [], error: errorMessage(err) };
  }
}
