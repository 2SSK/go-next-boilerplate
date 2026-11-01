"use client";

import { type FormEvent, useEffect, useState } from "react";
import { Loader2, Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api";
import { type Todo, todos as todosApi } from "@/lib/todos";
import { cn } from "@/lib/utils";

function messageOf(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  return "Could not reach the API. Is it running?";
}

/**
 * The todo demo: list, add, toggle, delete. Toggles and deletes update the list first
 * and roll back if the API refuses, so the UI never waits on the network.
 */
export function TodoList() {
  const [items, setItems] = useState<Todo[] | null>(null);
  const [title, setTitle] = useState("");
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    todosApi
      .list()
      .then(setItems)
      .catch((err: unknown) => {
        setItems([]);
        setError(messageOf(err));
      });
  }, []);

  const add = async (e: FormEvent<HTMLFormElement>): Promise<void> => {
    e.preventDefault();
    if (!title.trim() || adding) return;
    setAdding(true);
    setError(null);
    try {
      const created = await todosApi.create(title);
      setItems((prev) => [created, ...(prev ?? [])]);
      setTitle("");
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setAdding(false);
    }
  };

  /** Applies change locally, then calls the API; restores the snapshot on failure. */
  const optimistic = async (change: (prev: Todo[]) => Todo[], call: () => Promise<unknown>) => {
    const snapshot = items;
    setItems((prev) => change(prev ?? []));
    setError(null);
    try {
      await call();
    } catch (err) {
      setItems(snapshot);
      setError(messageOf(err));
    }
  };

  const toggle = (t: Todo): Promise<void> =>
    optimistic(
      (prev) => prev.map((x) => (x.id === t.id ? { ...x, completed: !t.completed } : x)),
      () => todosApi.update(t.id, { completed: !t.completed }),
    );

  const remove = (t: Todo): Promise<void> =>
    optimistic(
      (prev) => prev.filter((x) => x.id !== t.id),
      () => todosApi.remove(t.id),
    );

  const remaining = items?.filter((t) => !t.completed).length ?? 0;

  return (
    <div className="surface overflow-hidden">
      <form onSubmit={add} className="flex items-center gap-2 border-b p-3">
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="What needs doing?"
          aria-label="New todo"
          maxLength={200}
          className="h-10 border-transparent bg-transparent text-[15px] shadow-none focus-visible:border-transparent focus-visible:ring-0 dark:bg-transparent"
        />
        <Button
          type="submit"
          size="lg"
          disabled={!title.trim() || adding}
          className="rounded-full px-4"
        >
          {adding ? <Loader2 className="animate-spin" /> : <Plus />}
          Add
        </Button>
      </form>

      {error && (
        <p
          role="alert"
          className="bg-destructive/[0.06] text-destructive border-b px-5 py-2.5 text-sm"
        >
          {error}
        </p>
      )}

      {items === null ? (
        <div className="text-subtle flex justify-center py-12">
          <Loader2 className="size-5 animate-spin" />
        </div>
      ) : items.length === 0 ? (
        <p className="text-subtle py-12 text-center text-sm">
          Nothing here yet. Add your first todo.
        </p>
      ) : (
        <ul className="divide-y">
          {items.map((t) => (
            <li
              key={t.id}
              className="group animate-rise interactive hover:bg-foreground/[0.02] flex items-center gap-3 px-5 py-3"
            >
              <Checkbox
                checked={t.completed}
                onCheckedChange={() => void toggle(t)}
                aria-label={
                  t.completed ? `Mark "${t.title}" as not done` : `Mark "${t.title}" as done`
                }
              />
              <span
                className={cn(
                  "interactive flex-1 truncate text-[15px]",
                  t.completed && "text-subtle decoration-subtle/60 line-through",
                )}
              >
                {t.title}
              </span>
              <Button
                variant="ghost"
                size="icon-xs"
                onClick={() => void remove(t)}
                aria-label={`Delete "${t.title}"`}
                className="text-subtle hover:text-destructive opacity-0 group-hover:opacity-100 focus-visible:opacity-100 max-sm:opacity-100"
              >
                <X />
              </Button>
            </li>
          ))}
        </ul>
      )}

      {items !== null && items.length > 0 && (
        <div className="flex items-center justify-between border-t px-5 py-2.5">
          <span className="meta text-xs">
            {remaining} of {items.length} left
          </span>
        </div>
      )}
    </div>
  );
}
