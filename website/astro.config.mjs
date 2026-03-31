import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

export default defineConfig({
  site: "https://cloudmock.io",
  integrations: [
    starlight({
      title: "CloudMock",
      description: "Local AWS emulation for developers",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/neureaux/cloudmock",
        },
      ],
      customCss: [],
      head: [
        {
          tag: "meta",
          attrs: {
            name: "og:image",
            content: "/og-image.png",
          },
        },
      ],
      sidebar: [
        {
          label: "Getting Started",
          items: [
            { slug: "docs/getting-started/installation" },
            { slug: "docs/getting-started/first-request" },
            { slug: "docs/getting-started/with-your-stack" },
          ],
        },
        {
          label: "Services",
          autogenerate: { directory: "docs/services" },
        },
        {
          label: "Devtools",
          autogenerate: { directory: "docs/devtools" },
        },
        {
          label: "Language Guides",
          autogenerate: { directory: "docs/language-guides" },
        },
        {
          label: "Reference",
          autogenerate: { directory: "docs/reference" },
        },
      ],
    }),
  ],
});
