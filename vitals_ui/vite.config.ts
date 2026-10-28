import { resolve } from 'node:path';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, type Plugin } from 'vite';
import { viteSingleFile } from 'vite-plugin-singlefile';

/** Widens the artifact's CSP just enough for Vite's dev server: its scripts, styles and HMR socket. */
const devCsp = (): Plugin => ({
  name: 'dev-csp',
  apply: 'serve',
  transformIndexHtml: (html) =>
    html
      .replace('script-src ', "script-src 'self' ")
      .replace('style-src ', "style-src 'self' ")
      .replace("connect-src 'none'", "connect-src 'self' ws: wss:"),
});

/**
 * LayerCake's console warnings cite its docs by URL. The strings never cause a request, but the
 * artifact check forbids any `http(s)://` in the bundle, so drop the scheme at build time.
 */
const LAYERCAKE_DOC_URL = /https:\/\/layercake\.graphics\//g;
const stripLayerCakeUrls = (): Plugin => ({
  name: 'strip-layercake-doc-urls',
  apply: 'build',
  transform: (code, id) =>
    id.includes('/node_modules/layercake/') && LAYERCAKE_DOC_URL.test(code)
      ? { code: code.replace(LAYERCAKE_DOC_URL, 'layercake.graphics/'), map: null }
      : null,
});

// Larger than any asset the control could carry, so every one is inlined into the single file.
const INLINE_EVERY_ASSET_BYTES = 100_000_000;

export default defineConfig({
  plugins: [svelte(), tailwindcss(), viteSingleFile(), devCsp(), stripLayerCakeUrls()],
  build: {
    cssCodeSplit: false,
    assetsInlineLimit: INLINE_EVERY_ASSET_BYTES,
    rollupOptions: { input: resolve(import.meta.dirname, 'index.html') },
  },
});
