import { api, unwrap } from "./api";
import type { components } from "./api/schema";

/** Generated from the Todo schema in api/openapi.yaml; never edit by hand. */
export type Todo = components["schemas"]["Todo"];
export type TodoPatch = components["schemas"]["TodoPatch"];

/** Browser-side mutations. The page loads the list on the server (app/todos/page.tsx). */
export const todos = {
  create: (title: string): Promise<Todo> => unwrap(api.POST("/api/v1/todos", { body: { title } })),
  update: (id: number, patch: TodoPatch): Promise<Todo> =>
    unwrap(api.PATCH("/api/v1/todos/{id}", { params: { path: { id } }, body: patch })),
  remove: (id: number): Promise<void> =>
    unwrap(api.DELETE("/api/v1/todos/{id}", { params: { path: { id } } })).then(() => undefined),
};
