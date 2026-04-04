# CloudMock Platform: Next.js Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the customer-facing Next.js dashboard with Clerk auth, role-based views (Admin/Developer/Viewer), Stripe billing, and all management pages for the CloudMock SaaS platform.

**Architecture:** Next.js 15 app router deployed on Vercel. Clerk handles auth/SSO/org management. Stripe Customer Portal handles billing. All data operations go through the Go API at localhost:8080 (dev) or api.cloudmock.io (prod). Tailwind CSS for styling with CloudMock's green (#52b788) brand color.

**Tech Stack:** Next.js 15, React 19, Tailwind CSS 4, Clerk Next.js SDK, Stripe.js

**Spec:** `docs/superpowers/specs/2026-04-03-cloudmock-platform-design.md`

---

### Task 1: Next.js Project Setup

**Files:**
- Create: `apps/web/package.json`
- Create: `apps/web/next.config.ts`
- Create: `apps/web/tailwind.config.ts`
- Create: `apps/web/tsconfig.json`
- Create: `apps/web/app/layout.tsx`
- Create: `apps/web/app/globals.css`
- Create: `apps/web/lib/api.ts`
- Create: `apps/web/.env.local.example`

- [ ] **Step 1: Initialize Next.js project**

```bash
cd /Users/megan/cloudmock-platform
npx create-next-app@latest apps/web --typescript --tailwind --app --src-dir=false --import-alias="@/*" --no-eslint
```

- [ ] **Step 2: Install dependencies**

```bash
cd /Users/megan/cloudmock-platform/apps/web
npm install @clerk/nextjs
npm install stripe @stripe/stripe-js
```

- [ ] **Step 3: Create API client**

Create `apps/web/lib/api.ts` -- typed client that calls the Go API:

```typescript
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
  return res.json();
}

export interface App {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  endpoint: string;
  infra_type: string;
  status: string;
  created_at: string;
}

export interface APIKeyResponse {
  key: string;
  prefix: string;
  id: string;
  name: string;
  role: string;
}

export interface APIKeyListItem {
  id: string;
  prefix: string;
  name: string;
  role: string;
  last_used_at: string | null;
  created_at: string;
}

export interface UsageSummary {
  total_requests: number;
  period_start: string;
  period_end: string;
  apps: { app_id: string; app_name: string; request_count: number }[];
}

export const api = {
  apps: {
    list: (token: string) =>
      apiRequest<App[]>("/v1/apps", { headers: { Authorization: `Bearer ${token}` } }),
    get: (token: string, id: string) =>
      apiRequest<App>(`/v1/apps/${id}`, { headers: { Authorization: `Bearer ${token}` } }),
    create: (token: string, data: { name: string; infra_type: string }) =>
      apiRequest<App>("/v1/apps", {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: JSON.stringify(data),
      }),
    delete: (token: string, id: string) =>
      apiRequest<void>(`/v1/apps/${id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      }),
  },
  keys: {
    list: (token: string, appId: string) =>
      apiRequest<APIKeyListItem[]>(`/v1/apps/${appId}/keys`, {
        headers: { Authorization: `Bearer ${token}` },
      }),
    create: (token: string, appId: string, data: { name: string; role: string }) =>
      apiRequest<APIKeyResponse>(`/v1/apps/${appId}/keys`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: JSON.stringify(data),
      }),
    revoke: (token: string, appId: string, keyId: string) =>
      apiRequest<void>(`/v1/apps/${appId}/keys/${keyId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      }),
  },
  usage: {
    summary: (token: string) =>
      apiRequest<UsageSummary>("/v1/usage", { headers: { Authorization: `Bearer ${token}` } }),
  },
};
```

- [ ] **Step 4: Create env example**

```
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_...
CLERK_SECRET_KEY=sk_test_...
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_CLERK_SIGN_IN_URL=/sign-in
NEXT_PUBLIC_CLERK_SIGN_UP_URL=/sign-up
NEXT_PUBLIC_CLERK_AFTER_SIGN_IN_URL=/apps
NEXT_PUBLIC_CLERK_AFTER_SIGN_UP_URL=/apps
STRIPE_PUBLISHABLE_KEY=pk_test_...
```

- [ ] **Step 5: Configure root layout with Clerk**

```tsx
// apps/web/app/layout.tsx
import { ClerkProvider } from "@clerk/nextjs";
import "./globals.css";

export const metadata = {
  title: "CloudMock Platform",
  description: "Hosted AWS emulation for teams",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <ClerkProvider>
      <html lang="en" className="dark">
        <body className="bg-gray-950 text-gray-100 antialiased">{children}</body>
      </html>
    </ClerkProvider>
  );
}
```

- [ ] **Step 6: Set up Tailwind with brand colors**

Extend tailwind.config.ts to add CloudMock brand color `brand: "#52b788"`.

- [ ] **Step 7: Verify build**

```bash
cd /Users/megan/cloudmock-platform/apps/web && npm run build
```

- [ ] **Step 8: Commit**

```bash
cd /Users/megan/cloudmock-platform && git add -A && git commit -m "feat: Next.js project setup with Clerk, Stripe, Tailwind, and API client"
```

---

### Task 2: Auth Pages and Clerk Middleware

**Files:**
- Create: `apps/web/middleware.ts`
- Create: `apps/web/app/(auth)/sign-in/[[...sign-in]]/page.tsx`
- Create: `apps/web/app/(auth)/sign-up/[[...sign-up]]/page.tsx`

- [ ] **Step 1: Create Clerk middleware**

```typescript
// apps/web/middleware.ts
import { clerkMiddleware, createRouteMatcher } from "@clerk/nextjs/server";

const isPublicRoute = createRouteMatcher(["/sign-in(.*)", "/sign-up(.*)", "/"]);

export default clerkMiddleware(async (auth, request) => {
  if (!isPublicRoute(request)) {
    await auth.protect();
  }
});

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
```

- [ ] **Step 2: Create sign-in page**

```tsx
import { SignIn } from "@clerk/nextjs";

export default function SignInPage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <SignIn />
    </div>
  );
}
```

- [ ] **Step 3: Create sign-up page**

```tsx
import { SignUp } from "@clerk/nextjs";

export default function SignUpPage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <SignUp />
    </div>
  );
}
```

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: Clerk auth pages and middleware"
```

---

### Task 3: Dashboard Layout with Role-Based Sidebar

**Files:**
- Create: `apps/web/app/(dashboard)/layout.tsx`
- Create: `apps/web/components/sidebar.tsx`
- Create: `apps/web/lib/roles.ts`

- [ ] **Step 1: Create role helper**

```typescript
// apps/web/lib/roles.ts
export type Role = "admin" | "developer" | "viewer";

export function getRole(orgRole: string | undefined): Role {
  switch (orgRole) {
    case "org:admin": return "admin";
    case "org:developer": return "developer";
    default: return "viewer";
  }
}

export const sidebarItems = {
  admin: [
    { name: "Overview", href: "/overview", icon: "chart" },
    { name: "Apps", href: "/apps", icon: "grid" },
    { name: "API Keys", href: "/keys", icon: "key" },
    { name: "Usage & Billing", href: "/usage", icon: "credit-card" },
    { name: "Team", href: "/team", icon: "users" },
    { name: "Audit Log", href: "/audit", icon: "shield" },
    { name: "Settings", href: "/settings", icon: "settings" },
  ],
  developer: [
    { name: "Apps", href: "/apps", icon: "grid" },
    { name: "API Keys", href: "/keys", icon: "key" },
    { name: "Usage", href: "/usage", icon: "chart" },
    { name: "Team", href: "/team", icon: "users" },
  ],
  viewer: [
    { name: "Apps", href: "/apps", icon: "grid" },
    { name: "Usage", href: "/usage", icon: "chart" },
    { name: "Team", href: "/team", icon: "users" },
  ],
};
```

- [ ] **Step 2: Create sidebar component**

Sidebar with CloudMock logo, role-based navigation items, active state highlighting, Clerk UserButton at bottom. Uses `useOrganization` hook to get org role.

- [ ] **Step 3: Create dashboard layout**

Layout that wraps all (dashboard) pages with the sidebar on the left and content area on the right. Redirects to /apps by default.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: dashboard layout with role-based sidebar navigation"
```

---

### Task 4: Apps List Page (Developer Home)

**Files:**
- Create: `apps/web/app/(dashboard)/apps/page.tsx`
- Create: `apps/web/components/app-card.tsx`
- Create: `apps/web/components/create-app-dialog.tsx`

- [ ] **Step 1: Create app card component**

Card showing: app name, status badge (running/shared/dedicated), endpoint, request count, active services tags. Clickable to navigate to app detail.

- [ ] **Step 2: Create app dialog**

Modal dialog to create a new app. Fields: name (text input), infrastructure type (shared/dedicated radio). Calls api.apps.create on submit.

- [ ] **Step 3: Create apps list page**

Server component that fetches apps from the Go API using the Clerk session token. Shows grid of AppCards with a "+ New App" card. Uses `auth()` from Clerk to get the token.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: apps list page with create dialog and app cards"
```

---

### Task 5: App Detail Page

**Files:**
- Create: `apps/web/app/(dashboard)/apps/[id]/page.tsx`
- Create: `apps/web/app/(dashboard)/apps/[id]/keys/page.tsx`
- Create: `apps/web/components/copy-button.tsx`

- [ ] **Step 1: Create copy button component**

Small button that copies text to clipboard with "Copied!" feedback.

- [ ] **Step 2: Create app detail page**

Shows: app name + status badge, endpoint with copy button, quick-start snippet (`export AWS_ENDPOINT_URL=...`), active services tags. Tab navigation to sub-pages: Overview, Keys, Devtools, Snapshots, Settings.

- [ ] **Step 3: Create app keys page**

Lists API keys for this app. Create key button shows plaintext once in a modal. Revoke button. Each key shows: prefix, name, role, last used, created at.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: app detail page with API key management"
```

---

### Task 6: Admin Overview Page

**Files:**
- Create: `apps/web/app/(dashboard)/overview/page.tsx`
- Create: `apps/web/components/stat-card.tsx`
- Create: `apps/web/components/usage-chart.tsx`

- [ ] **Step 1: Create stat card component**

Card with label, large number, and sublabel. Used for: monthly requests, estimated cost, active apps, team size.

- [ ] **Step 2: Create usage chart**

Simple bar chart showing daily request volume for the past 30 days. Use CSS-only bars (no chart library needed for v1).

- [ ] **Step 3: Create admin overview page**

Grid of 4 stat cards at top, usage chart below, recent audit entries on the right. Only accessible to admin role -- redirect developers to /apps.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: admin overview page with usage stats and chart"
```

---

### Task 7: Usage, Billing, Team, Audit, and Settings Pages

**Files:**
- Create: `apps/web/app/(dashboard)/usage/page.tsx`
- Create: `apps/web/app/(dashboard)/billing/page.tsx`
- Create: `apps/web/app/(dashboard)/team/page.tsx`
- Create: `apps/web/app/(dashboard)/audit/page.tsx`
- Create: `apps/web/app/(dashboard)/settings/page.tsx`

- [ ] **Step 1: Create usage page**

Shows org-wide usage for current billing period. Per-app breakdown table. Estimated cost calculation ($0.50/10K after 1K free).

- [ ] **Step 2: Create billing page**

Stripe Customer Portal embed. Link to manage payment methods and view invoices. Shows current billing status.

- [ ] **Step 3: Create team page**

Uses Clerk's `<OrganizationProfile />` component for member management. Shows member list with roles. Admin can invite/change roles. Developer/Viewer see read-only list.

- [ ] **Step 4: Create audit log page (admin only)**

Paginated table of audit entries. Filters: action type, date range. CSV export button. Shows: timestamp, actor, action, resource, IP.

- [ ] **Step 5: Create settings page (admin only)**

Org name (from Clerk), data retention configuration (audit_log, request_log, state_snapshot with days input). Save button calls Go API.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: usage, billing, team, audit, and settings pages"
```

---

### Task 8: Landing Page and Final Polish

**Files:**
- Create: `apps/web/app/page.tsx`
- Create: `apps/web/app/(dashboard)/apps/[id]/loading.tsx`

- [ ] **Step 1: Create landing page**

Simple redirect to /apps if authenticated, /sign-in if not. Or minimal marketing page with "Get Started" CTA.

- [ ] **Step 2: Add loading states**

Loading skeletons for apps list, app detail, usage page.

- [ ] **Step 3: Verify full build**

```bash
cd /Users/megan/cloudmock-platform/apps/web && npm run build
```

- [ ] **Step 4: Final commit**

```bash
cd /Users/megan/cloudmock-platform && git add -A && git commit -m "feat: CloudMock Platform dashboard v1 complete"
```

---

## Self-Review

### Spec Coverage

| Spec Requirement | Task |
|-----------------|------|
| Clerk auth (sign-up, SSO, RBAC) | Tasks 1-2 |
| Role-based views (Admin/Developer/Viewer) | Task 3 |
| Apps list (developer home) | Task 4 |
| App detail + API keys | Task 5 |
| Admin overview (usage-first) | Task 6 |
| Usage & billing | Task 7 |
| Team management | Task 7 |
| Audit log (admin only) | Task 7 |
| Settings + data retention | Task 7 |
| Embedded devtools | Deferred (iframe to CloudMock :4500) |
| Snapshots management | Deferred (API exists, UI deferred) |
