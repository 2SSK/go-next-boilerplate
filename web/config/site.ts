/**
 * How the top navigation looks:
 * - "island": a floating frosted capsule. Suits products, marketing pages, personal apps.
 * - "bar": a full-width sticky bar with the app name. Suits platform, infra, and admin tools.
 */
export type NavbarStyle = "island" | "bar";

/** One place for the app's name, copy, and navigation. Edit this first. */
export const site: {
  readonly name: string;
  readonly description: string;
  readonly repo: string;
  readonly navbar: NavbarStyle;
} = {
  name: "Go Next Boilerplate",
  description: "A Go API and a Next.js frontend, wired together and ready to build on.",
  repo: "https://github.com/2SSK/go-next-boilerplate",
  navbar: "bar",
};

export interface NavItem {
  readonly name: string;
  readonly href: string;
}

/** Pages that need a signed-in user; proxy.ts sends anyone else to the login. */
export const protectedPaths: readonly string[] = [
  "/todos", // feature:todo
];

export const navigation: readonly NavItem[] = [
  { name: "Home", href: "/" },
  // feature:todo:begin
  { name: "Todos", href: "/todos" },
  // feature:todo:end
];
