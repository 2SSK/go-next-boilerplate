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

export const TODOS_PATH = "/api/v1/todos";

/** Browser-side mutations. The page loads the list on the server (app/todos/page.tsx). */
export const todos = {
  create: (title: string): Promise<Todo> =>
    api<Todo>(TODOS_PATH, { method: "POST", body: JSON.stringify({ title }) }),
  update: (id: number, patch: TodoPatch): Promise<Todo> =>
    api<Todo>(`${TODOS_PATH}/${id}`, { method: "PATCH", body: JSON.stringify(patch) }),
  remove: (id: number): Promise<void> => api<void>(`${TODOS_PATH}/${id}`, { method: "DELETE" }),
};
