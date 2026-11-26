import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["apple-touch-icon.png", "icon.svg"],
      manifest: {
        name: "Casa",
        short_name: "Casa",
        description: "Cámara, focos y homelab",
        lang: "es",
        start_url: "/",
        display: "standalone",
        background_color: "#0f1115",
        theme_color: "#0f1115",
        icons: [
          { src: "icon-192.png", sizes: "192x192", type: "image/png" },
          { src: "icon-512.png", sizes: "512x512", type: "image/png" },
          { src: "icon-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
        ],
      },
      workbox: {
        // La API y las imágenes NUNCA desde caché: el estado tiene que ser real
        navigateFallbackDenylist: [/^\/api\//, /^\/healthz/],
        runtimeCaching: [{ urlPattern: /\/api\//, handler: "NetworkOnly" }],
      },
    }),
  ],
  server: {
    // Desarrollo local: `go run .` en :8080 y `pnpm dev` aquí
    proxy: { "/api": "http://localhost:8080" },
  },
});
