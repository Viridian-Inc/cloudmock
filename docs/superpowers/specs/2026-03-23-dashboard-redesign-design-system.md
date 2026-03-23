# Dashboard Redesign: Design System & Layout Shell

**Date:** 2026-03-23
**Status:** Approved
**Sub-project:** 1 of 5

---

## Overview

Complete CSS rewrite applying the autotend design system (Figtree, brand navy/blue/teal palette, dark-first), plus rebuilt sidebar with section grouping and new base components. This replaces `styles/global.css` and restructures the layout shell.

## Design Tokens

```css
:root {
  /* Brand */
  --brand-dark: #0A1F44;
  --brand-blue: #097FF5;
  --brand-teal: #4AE5F8;
  --brand-orange: #F7711E;
  --brand-yellow: #FEC307;

  /* Backgrounds */
  --bg-primary: #080d17;
  --bg-secondary: #0c1322;
  --bg-tertiary: #111a2e;
  --bg-elevated: #162037;
  --bg-hover: rgba(9, 127, 245, 0.08);
  --bg-active: rgba(9, 127, 245, 0.15);

  /* Borders */
  --border-subtle: rgba(74, 229, 248, 0.06);
  --border-default: rgba(74, 229, 248, 0.12);
  --border-strong: rgba(74, 229, 248, 0.2);

  /* Text */
  --text-primary: #f0f2f5;
  --text-secondary: #8b95a5;
  --text-tertiary: #5a6577;
  --text-accent: #4AE5F8;

  /* Semantic */
  --success: #36d982;
  --warning: #fad065;
  --error: #ff4e5e;
  --info: #538eff;

  /* Radius */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;

  /* Fonts */
  --font-sans: 'Figtree', -apple-system, sans-serif;
  --font-mono: 'JetBrains Mono', ui-monospace, monospace;

  /* Layout */
  --sidebar-width: 220px;
}
```

## Sidebar

Grouped sections: Observe (Console, Services, Requests, Traces, Metrics), Respond (Incidents with badge, Regressions), Resources (DynamoDB, S3, SQS, Lambda, Cognito, IAM, Mail), bottom: Settings, Chaos.

Brand gradient logo, teal active state, subtle hover, section labels.

## Base Components

Rebuilt in the new design language:
- **SummaryCards** — stat cards with colored left border
- **DataTable** — sticky header, hover rows, sortable
- **Badge** — severity (critical/warning/info/success) + pill variant
- **Button** — primary (blue), ghost (border), danger, sizes sm/md
- **TabBar** — underline tabs with teal active
- **SearchBox** — Cmd+K style
- **Modal/Drawer** — dark overlay, elevated surface
- **StatusBadge** — HTTP method + status code coloring
- **ConfidenceBar** — teal fill bar with percentage

## File Changes

- Rewrite: `styles/global.css` — new tokens + all component styles
- Rewrite: `styles/console.css` — 3-panel layout with new tokens
- Rewrite: `components/Sidebar.tsx` — grouped sections, new styling
- Rewrite: `components/Header.tsx` — integrated into sidebar or minimal top bar
- Update: `components/SummaryCards.tsx` — match new card style
- Update: `components/FlameGraph.tsx` — match new colors
- Update: `App.tsx` — new layout structure
- Update: `main.tsx` — add Figtree font import
