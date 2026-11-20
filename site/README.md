# Nexus website

The landing page at [nexus.plu.nz](https://nexus.plu.nz/) explains Nexus and links to the latest release archive. It is a separate pnpm project from the app: SolidJS 1.9, TypeScript, Vite, the pinned `@plunz/storybook`, and Cloudflare Workers Static Assets. There are no runtime APIs, accounts, or analytics.

- [index.html](index.html) holds the static metadata.
- [src/index.tsx](src/index.tsx) renders the page and the illustration of the panel over a video.
- [src/locale.ts](src/locale.ts) holds the English, Portuguese, and Spanish copy. The language follows the browser, and an explicit choice is shared with other PLUNZ sites through the `plunz_locale` cookie ([src/locale-preference.ts](src/locale-preference.ts)).
- [src/style.css](src/style.css) owns the layout, mobile first down to 320 px.
- [public/favicon.svg](public/favicon.svg) is a copy of the app icon in `assets/`.

The download button points to `releases/latest/download/nexus-linux-x86_64.tar.gz`, which the [release workflow](../.github/workflows/release.yml) publishes on each `v*` tag. Keep the archive name and the install steps on the page in sync with [scripts/package.sh](../scripts/package.sh) and [scripts/install.sh](../scripts/install.sh).

```bash
cd site
pnpm install --frozen-lockfile
pnpm dev
pnpm check
pnpm deploy   # build and publish nexus.plu.nz to the PLUNZ Cloudflare account
```
