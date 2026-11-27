# Brand — Calango

## Logo

Original file: [logo-original.png](logo-original.png) (1254×1254). Reduced versions in `site/assets/`.

Reading of the style:
- **Cut-paper collage.** The lizard is made of pieces of paper with straight edges and irregular corners, slightly overlapping, with fiber texture.
- **Sticky-note background.** Overlapping yellow squares in slightly different tones, like a wall of ideas.
- **Liquid glass tile.** Everything inside a square with well-rounded corners, a translucent border, a white reflection at the top and yellow glow on the edges.
- **Warm, solar palette:** sticky-note yellow, lime/leaf/moss greens, an orange stripe, sand belly, clay rock, dark-brown eye.
- **Stance:** the lizard holds its head up, alert, on top of the rock. It represents an assistant that is awake and always on guard.

## Visual principles

1. **Handmade, premium finish.** Cut-paper shapes with soft shadows; glass only for highlights (the icon tile, main cards).
2. **Pieces that come together.** The central metaphor: pieces of paper (tasks, nodes, routines) joining to form the lizard. Use it in animations and diagrams.
3. **Warmth, not neon.** No purple, no generic AI gradient. Yellow, green and earth.
4. **Slight tilt.** Paper elements with a small rotation (−3° to 3°), never too aligned.

## Palette

See `packages/tokens/tokens.json`. Main colors: `postit #FDD838`, `lime #B6E021`, `leaf #6CC23A`, `moss #42912E`, `tangerine #FE9904`, `sand #F3CC85`, `clay #B47133`, `bark #4A2A15`, `paper #FFFBEF`, `ink #22180F`.

Contrast: text is always `ink` on light backgrounds and `paper` on `moss`/`bark`/`night`. Never white text on yellow.

## Typography

- Headings: **Bricolage Grotesque** 700–800, tight spacing, large size.
- Text: **Instrument Sans** 400–600.
- Code: **JetBrains Mono**.

## Motion

- **Settling paper:** an element enters with a small rotation and scale 0.96 → 1, curve `ease-paper` (overshoots slightly and comes back).
- **Staggering:** groups enter in sequence, 45 ms apart.
- **Single clock** in the product demos: one timeline drives every scene, allowing pause, skip and replay.
- `prefers-reduced-motion`: no translation or rotation; opacity only.

## Icon at small sizes

The website uses `site/assets/favicon.svg`: a flat moss lizard on post-it yellow, simplified for small rendering. Tray and notification variants remain pending.
