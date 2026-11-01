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

| Path                    | Holds                                                                 |
| ----------------------- | --------------------------------------------------------------------- |
| `app/globals.css`       | Every design token and the shared helper classes. Styling starts here |
| `app/layout.tsx`        | Fonts, theme provider, navbar, footer                                 |
| `config/site.ts`        | App name, description, repo link, navbar style, navigation            |
| `components/ui/`        | shadcn components. Add with `npx shadcn@latest add <name>`            |
| `components/`           | App components, kebab-case files (`api-status.tsx`)                   |
| `components/<feature>/` | Components owned by one feature                                       |
| `lib/api.ts`            | `api<T>()` fetch helper and `ApiError`                                |
| `lib/<feature>.ts`      | Typed API calls and types for one feature                             |

## Conventions

- **Server components by default.** Add `"use client"` only to the component that needs state,
  effects, or browser APIs, and keep that boundary as low in the tree as you can.
- **API calls go through `lib/api.ts`** with same-origin paths (`/api/v1/...`). Never hard-code
  the API host; `next.config.ts` forwards to `API_URL`. Each feature gets a `lib/<feature>.ts`
  whose types mirror the Go JSON exactly (camelCase fields).
- **Errors.** `api()` throws `ApiError` carrying the server's `error` message and status. Show
  it in the UI (`role="alert"`); don't swallow it.
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

Set `navbar` in `config/site.ts`:

- **`"bar"`**: a full-width bar stuck to the top, with the app mark and name on the left and
  links on the right. Use it for platform, infra, admin, and internal tools: anything people
  navigate repeatedly and densely.
- **`"island"`**: a frosted capsule floating at the top (`shadow-float`, `backdrop-blur`).
  Use it for products, landing pages, and personal sites, where the page is the point.

Both share the active-link style (a faint foreground tint) and the theme toggle. The `<html>`
element carries `data-navbar`, and `.page-top` reads it, so pages never hard-code top padding.

## Motion

- **Three durations on one curve** (`--ease-out-cubic`): `--duration-quick` 120ms (presses),
  `--duration-normal` 220ms (hovers, fades, theme switch), `--duration-slow` 400ms (reveals).
- **`.interactive`** gives any hoverable element the shared transition. Pair it with
  `active:scale-[0.98]` (buttons) or `active:scale-95` (icon buttons).
- **`.animate-rise`** fades blocks in from 8px below. Stagger page blocks by 60ms with
  `style={{ animationDelay }}`, about six steps at most.
- The theme switch cross-fades through the View Transitions API.
- `prefers-reduced-motion` reduces all motion to effectively nothing. Nothing may depend on an
  animation finishing.

## Components and icons

- **Icons:** lucide only, `size-4` in UI and `size-3.5` in dense rows. Icon-only buttons get an
  `aria-label` and a `title`. No emoji as icons.
- **Status:** a pill with a dot (`components/api-status.tsx`), where the dot colour is the state.
  Use the ping animation only for a live, healthy state.
- **Empty, loading, and error states** are designed, not left blank: a centered `subtle` line
  of text for empty, a `Loader2` spinner for loading, and an inline `role="alert"` strip for errors.
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
- Introduce another icon set, font, animation library, or CSS-in-JS.
