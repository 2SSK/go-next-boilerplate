<!-- BEGIN:nextjs-agent-rules -->

## This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

# Principles

These apply to the whole repo (see `../AGENTS.md`):

- Do not preserve backward compatibility. Remove obsolete paths instead of adding compatibility layers, fallbacks, or migrations.
- Choose the simplest implementation that fully meets the current requirements. Avoid speculative abstractions, configurations, and indirections.
- Grow the system in layers. Start from the smallest version that works to end, and add each new capability on top of a product that already works. Never trade a working product for unfinished complexity.
- Keep components modular and concerns clearly separated.
- Prefer established, well-maintained libraries when they reduce overall complexity or improve reliaability. Do not reimplement common functionality without a clear reason.
- Lean on the dependencies already in the project before writing your own implementation or adding packages. Do not assume a library lacks a capability without checking its documentation and types.
- Make architectural decisions for the long term. Do not acccept a stopgap that only works for now and is meant to be replaced later.

# Web guide

Next.js App Router, React 19, TypeScript (strict), Tailwind v4, shadcn/ui on Base UI
(`base-nova` style), lucide icons, next-themes. Run `npm run lint`, `npm run typecheck`, and
`npm run format:check` (or `make lint` from the root) before finishing.

## Layout

| Path                    | Holds                                                                    |
| ----------------------- | ------------------------------------------------------------------------ |
| `app/globals.css`       | Every design token and the shared helper classes. Styling starts here    |
| `postcss.config.mjs`    | Tailwind v4 via `@tailwindcss/postcss`, the setup Next.js documents      |
| `app/layout.tsx`        | Fonts, theme provider, navbar, footer: everything shared by pages        |
| `config/site.ts`        | App name, description, repo link, navbar style, navigation               |
| `components/ui/`        | shadcn components. Add with `npx shadcn@latest add <name>`               |
| `components/`           | App components, kebab-case files (`api-status.tsx`)                      |
| `components/navbar.tsx` | Server-rendered navbar; `nav-links.tsx` is its only client part          |
| `components/<feature>/` | Components owned by one feature                                          |
| `lib/api/schema.ts`     | Types generated from `../api/openapi.yaml` (`make generate`); never edit |
| `lib/api.ts`            | Browser client `api`, `unwrap()`, `ApiError`, `errorMessage()`           |
| `lib/api-server.ts`     | Server-only client `serverApi` for server components                     |
| `lib/api-url.ts`        | `API_URL`, shared by `next.config.ts` and `serverApi()`                  |
| `lib/<feature>.ts`      | One feature's types (from the schema) and browser-side calls             |
| `e2e/`                  | Playwright tests against the running stack (`make up`, `make e2e`)       |

## Conventions

- **Server components by default.** Add `"use client"` only to the component that needs state,
  effects, or browser APIs, and keep that boundary as low in the tree as you can.
- **API calls are typed by the contract.** Types come from `lib/api/schema.ts`, generated from
  `api/openapi.yaml`; never hand-write a type the API already defines (`type Todo =
components["schemas"]["Todo"]`). Server components call `serverApi.GET(...)`; client
  components call `api.POST(...)` and friends with same-origin paths, which `next.config.ts`
  forwards to `API_URL`. Wrap calls in `unwrap()` to get the data or a thrown `ApiError`. Never
  hard-code the API host. An endpoint that doesn't exist in the spec is a type error.
- **Errors.** `unwrap()` throws `ApiError` with the server's message, status, and `code` (the
  contract's machine-readable reason). Branch on `code`, show `message` in the UI
  (`role="alert"`), and never swallow it.
- **Mutations.** Update local state first and roll back if the request fails (see
  `components/todos/todo-list.tsx`). Creates wait for the server, because the server assigns the ID.
- **Types.** No `any`. Use `interface` for object shapes, `type` for unions. Props are
  `Readonly<...>`. Use `import type` for type-only imports.
- **Imports.** Use the `@/` alias, never long relative paths. Order: framework, third-party,
  `@/` internal, then CSS.
- **shadcn.** Restyle through tokens in `globals.css`, not by editing `components/ui/`. Edit a
  ui component only for a behavior change, and expect `shadcn add --overwrite` to undo it.
- **Removable features.** Feature-only lines in shared files carry `feature:<name>` markers
  (see `../AGENTS.md`). In JSX, put `{/* ... */}` marker comments on their own lines.

## Rendering and performance

The app must feel instant. Four things make it so; every rule below protects one of them:

1. **The chrome never re-renders.** The navbar and footer live in the root layout, which stays
   mounted while only the page under it changes. Only the small `NavLinks` part runs in the
   browser.
2. **Pages are ready before the click.** Each route has a prebuilt static shell, links prefetch
   it, and page data loads on the server and streams in. There is no spinner after a page
   appears.
3. **Nothing moves that shouldn't.** The scrollbar's space is always reserved, skeletons match
   the final layout, and fonts and images hold their size, so the layout never shifts.
4. **One calm animation per page change,** which never delays navigation. The URL, tab title,
   and content change in the same frame.

### Structure

- **Shared chrome lives in a layout, never in a page.** The navbar, footer, and anything else
  shared across pages render from `app/layout.tsx`. Chrome for one section of the app goes in
  that section's nested `layout.tsx` (use a route group `(name)/` if the URL shouldn't change).
  Pages render only their own content, with a single `<main>` as their root.
- **No `template.tsx`.** A template gets a new key on every navigation, so everything under it
  remounts, loses its state, and re-runs its effects. That is exactly what layouts avoid. Use a
  layout.
- **Client islands are leaves.** Server components by default. When something needs the URL,
  state, or the DOM, isolate that piece and keep the rest on the server: `Navbar` is a server
  component, and `NavLinks` is the only part that calls `usePathname()`, so a navigation
  re-renders a few links and nothing else. Never put `"use client"` on a layout or a page.

### Data

- **Load page data on the server and stream it.** An async server component calls
  `serverApi()`, sits inside `<Suspense>` with a skeleton fallback, and passes the data as props
  to the client component that handles interaction (`app/todos/page.tsx`). Never fetch a page's
  initial data in `useEffect`; that adds a round trip after hydration.
- **Every route has a static shell** (Cache Components / Partial Prerendering). Uncached data
  and runtime APIs (`cookies()`, `headers()`, `searchParams`) go inside `<Suspense>`. Data that
  is the same for every visitor gets `"use cache"`, with `cacheLife`/`cacheTag` for freshness.
- **Titles are static metadata.** Use `export const metadata = { title: "..." }` so the title is
  part of the shell and switches with the URL. Reach for `generateMetadata` only when the title
  depends on data, and keep that data fast or cached: metadata that waits on slow data streams
  in late, leaving the tab showing the URL in the meantime.
- **Mutations update the screen first** and roll back on failure (see Conventions), so actions
  never wait on the network.

### Navigation

- **Internal links use `<Link>`**, which prefetches the target's shell when the link enters the
  viewport. Don't use `<a>` or `router.push` for plain navigation. For a link whose target
  depends on its URL's params or search params, `prefetch={true}` also prefetches that cached
  content.
- **Pages stay mounted when hidden.** Next.js keeps recently visited routes alive, hidden, via
  React `<Activity>`, so state survives Back. Effects are cleaned up while a route is hidden and
  re-run when it comes back: every effect must clean up fully (timers, listeners, requests).
  Reset state explicitly (for example after a form submits); never rely on unmount.
- **Navigation never waits on an animation.** On a fresh load, a page's blocks rise in with a
  stagger (`.animate-enter`). After the first client navigation, `RouteMotion` sets
  `<html data-navigated>`: the stagger turns off, and the whole incoming page rises in once
  (`[data-navigated] main`, 220ms). It is a plain CSS animation on the already-rendered page, so
  the router commits immediately.
- **Don't wrap routes in React's `<ViewTransition>`**, in a layout or in each page. A view
  transition freezes the page until it finishes, and React holds the next navigation behind it.
  We measured the result: a click during that window fell back to a full page reload, and the
  tab title went blank for about a second. `<ViewTransition default="none">` is fine for one
  contained element inside a page, never for whole routes.

### Layout stability

- **`html` has `scrollbar-gutter: stable`**, so short pages (no scrollbar) and long pages share
  one centre line. Without it, the whole layout jumps 6px sideways between them.
- **Skeletons are the real layout with placeholders** (`TodoListSkeleton`): the same container,
  row height, and padding, so nothing moves when data streams in.
- **Fonts come from `next/font`**, which self-hosts them and sizes a fallback to match, so text
  doesn't reflow when the font loads. **Images use `next/image`** with `width`/`height` (or
  `fill` in a sized box), so their space is reserved before they load.
- **Never insert content above what's on screen** after load (banners, late-loading notices).
  Reserve the space up front or place it below.

### Runtime

- **The React Compiler memoizes for you.** Don't write `useMemo`, `useCallback`, or `memo`.
  Follow the Rules of React; the `react-hooks` lint rules enforce them (for example, no JSX
  inside `try/catch`: do the fallible work first, then render).
- **Background work only while visible.** Polling stops while the tab is hidden and runs at once
  when it returns (`components/api-status.tsx`).
- **Watch the bundle.** Prefer libraries that run in server components and never ship to the
  browser. Icons come from lucide (tree-shaken per icon). A new client dependency needs a reason.
- **Keep the Tailwind setup** (`postcss.config.mjs` with `@tailwindcss/postcss`). The
  `@tailwindcss/turbopack` loader that `create-next-app` offers stopped rebuilding CSS in dev, so
  edits to `globals.css` silently never showed up.

### Budget and how to check it

Measured on a production build (`npm run build && npx next start`); dev mode compiles routes on
first visit and is no guide to speed.

| Measure                                      | Budget                     | Measured |
| -------------------------------------------- | -------------------------- | -------- |
| Click to new page (URL, title, content)      | under 50ms once prefetched | 6–13ms   |
| Time with no tab title during navigation     | 0 frames                   | 0        |
| Static page from the server (`/`)            | under 20ms                 | 8ms      |
| Streamed page with data (`/todos`, complete) | under 100ms on a local API | ~46ms    |
| Horizontal shift between pages               | 0px                        | 0px      |
| JavaScript for a full visit (transferred)    | under 200KB                | 164KB    |

Before finishing a change that touches pages, layouts, or data loading:

- [ ] `npm run build`: every page is `○` (static) or `◐` (partial) in the route table. A `ƒ`
      (fully dynamic) page needs a reason.
- [ ] Click between pages quickly, several times in a row: no full page reload, no blank tab
      title, no sideways jump.
- [ ] Load a data page with the API stopped: it renders with an inline error, not a crash.
- [ ] Check light and dark mode, and a 375px-wide viewport.

# Design system

The look comes from one idea: **quiet surfaces, one accent, and motion that never draws
attention to itself.** It mirrors the author's desktop shell and site (sskbtw.xyz). Build
anything new from the tokens and helper classes below. If a design needs a new token, add it to
`globals.css` in both themes; never hard-code it in a component.

## Principles

1. **Content first.** Chrome stays quiet: hairline borders, soft shadows, no gradients, no
   decorative illustration. A screen should read as text and data on paper.
2. **One accent.** `primary` (blue) marks the single most important action and the current
   state. Never put two primary buttons side by side; the second is `outline` or `ghost`.
3. **Colour means state.** `success` and `destructive` appear only for status (healthy, error,
   delete). They are never decoration.
4. **Mono is for data.** Routes, IDs, counts, timestamps, versions, and code use JetBrains
   Mono. Everything people read as prose uses Inter.
5. **Motion confirms; it doesn't perform.** Short, eased, small distances. Nothing bounces,
   spins (except loaders), or loops for attention.
6. **Both themes are first-class.** Check every change in light and dark.

## Colour tokens

Use the semantic Tailwind names (`bg-card`, `text-muted-foreground`), never raw colours.

| Token                            | Light     | Dark      | Use                                          |
| -------------------------------- | --------- | --------- | -------------------------------------------- |
| `background`                     | `#f7f7f9` | `#1a1b26` | Page                                         |
| `card` / `popover`               | `#ffffff` | `#1f2335` | Raised surfaces                              |
| `foreground`                     | `#1a1b26` | `#c0caf5` | Primary text                                 |
| `muted-foreground`               | `#565a6e` | `#a9b1d6` | Secondary text, descriptions                 |
| `subtle`                         | `#858aa0` | `#737aa2` | Tertiary: labels, meta, placeholders, icons  |
| `primary`                        | `#2a6fdb` | `#7aa2f7` | The accent: main action, active state, focus |
| `secondary` / `muted` / `accent` | `#eeeff3` | `#292e42` | Quiet fills                                  |
| `border` / `input`               | `#e5e6ec` | `#2c3047` | Hairlines                                    |
| `success`                        | `#3d7a2f` | `#9ece6a` | Healthy, done                                |
| `destructive`                    | `#d6335a` | `#f7768e` | Errors, delete                               |

Light mode is neutral paper with Tokyo Night Day accents; dark mode is Tokyo Night. For tints,
use opacity on foreground, not a new grey: `bg-foreground/[0.05]` for chips and active nav
items, `/[0.02]` for row hover, and `bg-destructive/[0.06]` behind error text.

## Typography

- **Inter** (`font-sans`) for text, with `cv11` and `ss01` features on. **JetBrains Mono**
  (`font-mono`) for data.
- Headings are semibold with tight tracking, already set in the base layer.
- **Scale:**
  - Hero: `text-[40px] sm:text-[56px] tracking-[-0.03em] leading-[1.08]`
  - Page title: `text-[32px] tracking-[-0.02em]`
  - Lead text: 17–19px in `muted-foreground`
  - Body and list rows: 15px
  - UI text (buttons, nav): `text-sm`
- **`.label`**: section heading. 12px, semibold, uppercase, 0.08em tracking, `subtle`.
- **`.meta`**: data text. 13px mono, tabular numbers, `subtle`.

## Surfaces and shape

- **`.surface`**: the standard card: `rounded-2xl`, hairline border, `bg-card`, `shadow-card`.
  Lists inside it use `divide-y`, with rows padded `px-5 py-3`.
- **Two elevations only.** `shadow-card` for content; `shadow-float` for things above content
  (island navbar, popovers). The dark theme replaces drop shadows with an inset highlight.
- **Radius.** Cards `2xl`, inputs and menus `lg`, pills and chips `full`. Buttons in hero areas
  are `rounded-full`; buttons inside dense UI keep the shadcn default.
- **Layout.** One column: `.container-page` (`max-w-3xl`, 20–24px gutters). Start every page
  with `.page-top`, which sets the right space for the chosen navbar style. Sections are spaced
  `mt-12` to `mt-16`.

## Navbar

Switch styles with one line in `config/site.ts`: `navbar: "bar"` or `navbar: "island"`.

- **`"bar"`**: a full-width bar stuck to the top, with the app mark and name on the left and
  links on the right. Use it for platform, infra, admin, and internal tools: anything people
  navigate repeatedly and densely.
- **`"island"`**: a frosted capsule floating at the top (`shadow-float`, `backdrop-blur`).
  Use it for products, landing pages, and personal sites, where the page is the point.

Both share the active-link style (a faint foreground tint) and the theme toggle. The `<html>`
element carries `data-navbar`, and `.page-top` reads it, so pages never hard-code top padding.
Either style renders once from the root layout and stays mounted across navigations.

## Motion

- **Three durations on one curve** (`--ease-out-cubic`): `--duration-quick` 120ms (presses),
  `--duration-normal` 220ms (hovers, fades, theme switch), `--duration-slow` 400ms (reveals).
- **`.interactive`** gives any hoverable element the shared transition. Pair it with
  `active:scale-[0.98]` (buttons) or `active:scale-95` (icon buttons).
- **`.animate-enter`** is a page's entrance on a fresh load: blocks fade in from 8px below,
  staggered by 60ms with `style={{ animationDelay }}`, about six steps at most. It switches
  off after the first client navigation.
- **`.animate-rise`** is the same motion for something that appears inside a page, like a new
  list row. It always plays.
- **Route changes**: the incoming page rises in over 220ms (`[data-navigated] main` in
  `globals.css`), and the outgoing one is simply replaced. The navbar and footer live in the
  layout and never move. See "Navigation" under "Rendering and performance" for why this isn't
  a view transition.
- **The theme switch** cross-fades through the View Transitions API (220ms). The toggle applies
  the theme class to `<html>` synchronously inside the transition; next-themes alone would apply
  it in an effect, after the browser's "after" snapshot. Scoped by `data-theme-switch`.
- `prefers-reduced-motion` reduces all motion to effectively nothing. Nothing may depend on an
  animation finishing.

## Components and icons

- **Icons:** lucide only, `size-4` in UI and `size-3.5` in dense rows. Icon-only buttons get an
  `aria-label` and a `title`. No emoji as icons.
- **Status:** a pill with a dot (`components/api-status.tsx`), where the dot colour is the state.
  Use the ping animation only for a live, healthy state.
- **Empty, loading, and error states** are designed, not left blank. Empty: a centered
  `subtle` line of text. Loading page data: a skeleton that mirrors the final layout
  (`TodoListSkeleton`), so nothing jumps when the data streams in. An action in flight: a
  `Loader2` spinner in its button. Errors: an inline `role="alert"` strip.
- Hover-revealed actions (like delete) must stay visible on touch screens (`max-sm:opacity-100`)
  and when focused by keyboard (`focus-visible:opacity-100`).

## Accessibility

- Keep the focus rings shadcn provides (`ring`); never remove an outline without replacing it.
- Live regions: `role="status"` for passive updates, `role="alert"` for errors.
- Text colours meet contrast on their surfaces in both themes; `subtle` is for non-essential
  text only.
- `aria-current="page"` on the active nav link.

## Don't

- Add a second accent colour, gradients, coloured backgrounds, or heavy shadows.
- Use raw Tailwind palette colours (`text-gray-500`, `bg-blue-600`) or hex values in components.
- Add top padding to a page by hand; use `.page-top`.
- Render the navbar or footer inside a page.
- Add a `template.tsx`, fetch a page's initial data in `useEffect`, or wrap routes in
  `<ViewTransition>`. Each one makes navigation slower or less reliable (see "Rendering and
  performance").
- Introduce another icon set, font, animation library, or CSS-in-JS.
