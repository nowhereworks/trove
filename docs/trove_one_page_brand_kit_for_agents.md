# Trove — One-Page Brand Kit for AI Agents

## Purpose
Trove is a self-hosted software registry for AI agents, developer tools, prompts, workflows, models, and runtime artifacts. The website and product UI must feel secure, practical, technical, organized, durable, trustworthy, and developer-first. Every visual decision should reinforce that Trove is infrastructure software: reliable, self-hosted, versioned, controlled, and built for serious engineering teams.

## Visual Direction
Use a modern developer-infrastructure aesthetic: dark mode first, clean geometric shapes, strong contrast, restrained accent color, generous whitespace, and precise alignment. Avoid playful, cute, fantasy, crypto, generic cloud, or consumer SaaS styling. The brand should feel closer to a secure artifact registry, package manager, private catalog, or infrastructure control plane than a marketing-heavy SaaS landing page.

## Logo System
Use a simple geometric symbol with an optional `Trove` wordmark. Preferred visual metaphors are secure package, archive drawer, stacked artifacts, discovery/search over package, or minimal industrial crate. The symbol must remain legible at favicon size and work in monochrome.

**Primary lockup:** symbol + `Trove` wordmark.  
**Secondary mark:** symbol only for app icon, favicon, GitHub avatar, CLI/docs icon.  
**Monochrome:** off-white mark on dark graphite.  
**Clear space:** minimum `0.5×` symbol width around all sides.  
**Minimum sizes:** favicon `16×16`, UI icon `20×20`, app icon `32×32`, full logo `120px` wide.

## Color Palette
Use dark graphite as the dominant base, off-white for primary content, muted neutral for secondary content, and one restrained accent color per composition.

| Token | Hex | Usage |
|---|---:|---|
| `--color-bg` | `#0E1116` | Main page background |
| `--color-surface` | `#161B22` | Cards, panels, nav, icon tiles |
| `--color-border` | `#2A3340` | Subtle dividers, outlines, separators |
| `--color-text` | `#F2F3F5` | Primary text, wordmark, headings |
| `--color-muted` | `#A6A39A` | Body copy, captions, secondary labels |
| `--color-accent` | `#E0AE3A` | Primary accent: CTAs, highlights, logo details |
| `--color-accent-blue` | `#4F6D8F` | Alternate accent for catalog/archive sections |
| `--color-accent-green` | `#7F9970` | Alternate accent for versioned/stacked artifact sections |
| `--color-accent-orange` | `#F08C2B` | Alternate accent for search/discovery sections |

**Usage ratio:** 70% dark neutrals, 20% off-white/light neutral, 10% accent. Never mix multiple accent colors in the same logo lockup or primary section unless there is a clear semantic reason.

## Typography
Use `Inter` as the primary font. Fallback stack: `Inter, Manrope, IBM Plex Sans, Segoe UI, Arial, sans-serif`.

| Role | Weight | Size Guidance | Style |
|---|---:|---:|---|
| Wordmark | `800` | visual, not body-scaled | Title case, tight tracking `-0.01em` to `-0.02em` |
| Hero heading | `700–800` | `48–72px` desktop | Clean, confident, minimal |
| Section heading | `700` | `28–40px` | Title case or uppercase sparingly |
| Label / eyebrow | `600` | `12–14px` | Uppercase, tracking `0.10em–0.14em` |
| Body | `400–500` | `16–18px` | Line height `1.5–1.6` |
| Caption / metadata | `500` | `12–14px` | Muted color, tracking `0.04em–0.08em` |

## Layout Rules
Use an `8px` spacing system. Prefer grid-aligned layouts, strong vertical rhythm, and compact technical sections. Use subtle 1px borders instead of heavy shadows. Rounded corners should be present but not playful: `12–24px` for cards, `16–24px` for app-icon mockups, `4–8px` for small internal icon geometry.

Recommended website section behavior:
- Hero: dark background, strong wordmark or product headline, one accent CTA.
- Feature cards: graphite surfaces, 1px borders, concise technical copy.
- Code/CLI areas: terminal-inspired dark panels with off-white text and amber highlights.
- Docs links and metadata: muted text, compact spacing, high scanability.

## Iconography
Supporting icons should be geometric line icons with consistent stroke weight. Preferred motifs: shield, lock, package, archive drawer, layers, cube, semantic version tag, magnifier, terminal, code brackets, checkmark. Use accent color for icon strokes and off-white or muted text for labels. Avoid filled emoji-like icons, glossy icons, mascots, or illustrative clutter.

## Component Styling
Buttons should be compact, technical, and high-contrast. Primary CTA uses amber accent on dark or dark text on amber. Secondary buttons use transparent/surface backgrounds with subtle borders. Cards use `#161B22`, `#2A3340` borders, `#F2F3F5` headings, and `#A6A39A` body text. Badges may use muted surfaces with accent text for states like `self-hosted`, `immutable`, `OIDC`, `RBAC`, `CLI-first`, and `PostgreSQL`.

## Voice and Copy
Copy should be direct, technical, and concrete. Prefer phrases like “Self-hosted artifact registry”, “Immutable packages”, “CLI-first workflows”, “RBAC and OIDC”, “Private by default”, and “Versioned runtime artifacts”. Avoid vague marketing language like “revolutionary”, “magical”, “next-gen”, or “AI-powered platform” unless technically justified.

## Do / Don’t
**Do:** use dark mode, restrained accents, geometric symbols, strong whitespace, technical labels, concise copy, and monochrome-safe assets.  
**Don’t:** use fantasy treasure chests, cartoon mascots, crypto/gem shapes, generic cloud logos, bright gradients, excessive shadows, playful rounded typography, or multi-color decorative palettes.

## CSS Starter Tokens
```css
:root {
  --color-bg: #0E1116;
  --color-surface: #161B22;
  --color-border: #2A3340;
  --color-text: #F2F3F5;
  --color-muted: #A6A39A;
  --color-accent: #E0AE3A;
  --color-accent-blue: #4F6D8F;
  --color-accent-green: #7F9970;
  --color-accent-orange: #F08C2B;

  --font-sans: Inter, Manrope, "IBM Plex Sans", "Segoe UI", Arial, sans-serif;
  --radius-sm: 8px;
  --radius-md: 16px;
  --radius-lg: 24px;
  --space-unit: 8px;
}
```

## Implementation Rule
When in doubt, optimize for a secure developer infrastructure product: dark, precise, durable, practical, self-hosted, and easy to scan.

