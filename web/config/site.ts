/** One place for the app's name, copy, and navigation. Edit this first. */
export const site = {
  name: "Boilerplate",
  description: "A Go API and a Next.js frontend, wired together and ready to build on.",
  repo: "https://github.com/2SSK/go-next-boilerplate",
} as const;

export interface NavItem {
  readonly name: string;
  readonly href: string;
}

export const navigation: readonly NavItem[] = [
  { name: "Home", href: "/" },
  // feature:todo:begin
  { name: "Todos", href: "/todos" },
  // feature:todo:end
];
