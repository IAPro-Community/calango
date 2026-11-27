# ADR 0025 — Desktop UI kit in `src/lib` with shadcn and Tailwind

Status: accepted · 2026-11-26 · issue #180 · implements the Figma Design System V1 (#175) in code.

## Context

The desktop renderer is styled by one hand-written `styles.css` over the `--calango-*` tokens. The Figma Design System V1 defines a component library (`Button`, `Badge`, `NodePill`, `NavItem`, `ApprovalCard`, `StatCard`), a dark and a light ("solar") theme, a type scale and a spacing scale. The owner wants these components in code, kept apart so they can later become their own package, built with shadcn/ui and imported through `@`.

## Decision

1. **Location.** The kit lives in `apps/desktop/src/lib` and does not import from the renderer. `@` resolves to `src/lib` in Vite, Vitest and `tsconfig.renderer.json`. `components.json` points the shadcn CLI at it.
2. **shadcn/ui on Tailwind 4.** Components follow the shadcn pattern (`cva` variants, `cn` with `tailwind-merge`, `asChild` through `@radix-ui/react-slot`), restyled to the Figma variants. Tailwind runs through `@tailwindcss/vite`.
3. **Tokens stay the source.** Brand primitives keep coming from `packages/tokens` (`--calango-*`). `src/lib/styles/theme.css` adds the semantic layer (`--cl-*`) for the two themes, mirroring the Figma `Color` and `Color Light` collections and `site/index.html`, and maps it into Tailwind with `@theme inline`. Components use only those names, never raw colors. The danger text colour is adjusted per theme to keep WCAG AA (Figma's `#d9483b` stays on borders); `contrast.test.ts` checks every text pair in both themes.
4. **No preflight yet.** Tailwind is imported without its CSS reset, and `styles.css` moves into a `legacy` cascade layer below the utilities. The current screens render as before, and kit components win over the old element rules. The reset can come in when every screen uses the kit.
5. **Fonts are bundled.** Bricolage Grotesque, Instrument Sans and JetBrains Mono come from `@fontsource-variable` (OFL-1.1) and ship with the renderer, so the `font-src 'self'` CSP holds and the app works offline.
6. **Catalog.** `catalog.html` shows every component in both themes under `vite` dev. The production build bundles only `index.html`.

## Consequences

- New runtime dependencies: `class-variance-authority`, `clsx`, `tailwind-merge`, `@radix-ui/react-slot` and the three font packages; `tailwindcss` and `@tailwindcss/vite` as dev dependencies.
- The semantic theme values that are not primitives (for example the night text `#fff4da`) live in `theme.css` for now. Moving them into `packages/tokens/tokens.json` as themed tokens fits better once the Dart theme generator (#171) settles, so both apps read the same values.
- Screens move to the kit in a follow-up change (#180), after the open renderer PRs land.
