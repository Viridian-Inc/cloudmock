export type Role = "admin" | "developer" | "viewer";

export function getRole(orgRole: string | undefined): Role {
  switch (orgRole) {
    case "org:admin":
      return "admin";
    case "org:developer":
      return "developer";
    default:
      return "viewer";
  }
}

export const sidebarItems = {
  admin: [
    { name: "Overview", href: "/overview" },
    { name: "Apps", href: "/apps" },
    { name: "API Keys", href: "/keys" },
    { name: "Usage & Billing", href: "/usage" },
    { name: "Team", href: "/team" },
    { name: "Audit Log", href: "/audit" },
    { name: "Settings", href: "/settings" },
  ],
  developer: [
    { name: "Apps", href: "/apps" },
    { name: "API Keys", href: "/keys" },
    { name: "Usage", href: "/usage" },
    { name: "Team", href: "/team" },
  ],
  viewer: [
    { name: "Apps", href: "/apps" },
    { name: "Usage", href: "/usage" },
    { name: "Team", href: "/team" },
  ],
};
