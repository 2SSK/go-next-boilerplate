import { api } from "./api";

/** Mirrors todo.Todo in internal/todo/todo.go. */
export interface Todo {
  id: number;
  title: string;
  completed: boolean;
  createdAt: string;
  updatedAt: string;
}

export type TodoPatch = Partial<Pick<Todo, "title" | "completed">>;

const BASE = "/api/v1/todos";

export const todos = {
  list: (): Promise<Todo[]> => api<Todo[]>(BASE, { cache: "no-store" }),
  create: (title: string): Promise<Todo> =>
    api<Todo>(BASE, { method: "POST", body: JSON.stringify({ title }) }),
  update: (id: number, patch: TodoPatch): Promise<Todo> =>
    api<Todo>(`${BASE}/${id}`, { method: "PATCH", body: JSON.stringify(patch) }),
  remove: (id: number): Promise<void> => api<void>(`${BASE}/${id}`, { method: "DELETE" }),
};
