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
      sidebar: [
        {
          label: "Getting Started",
          items: [
            { slug: "getting-started/installation" },
            { slug: "getting-started/first-request" },
            { slug: "getting-started/with-your-stack" },
          ],
        },
        {
          label: "Services",
          autogenerate: { directory: "services" },
        },
        {
          label: "Devtools",
          autogenerate: { directory: "devtools" },
        },
        {
          label: "Language Guides",
          autogenerate: { directory: "language-guides" },
        },
        {
          label: "Reference",
          autogenerate: { directory: "reference" },
        },
      ],
    }),
  ],
});
