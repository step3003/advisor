import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

// В dev API-запросы (/api) проксируются на бэкенд advisord (по умолчанию :8080).
// Переопределяется переменной окружения ADVISOR_API_URL.
const apiTarget = process.env.ADVISOR_API_URL || "http://localhost:8080";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      manifest: {
        name: "Advisor — финансовый планировщик",
        short_name: "Advisor",
        description: "Личный финансовый учёт: план/факт, статистика, графики",
        lang: "ru",
        start_url: "/",
        display: "standalone",
        background_color: "#ffffff",
        theme_color: "#1971c2",
        icons: [
          { src: "icon-192.png", sizes: "192x192", type: "image/png" },
          { src: "icon-512.png", sizes: "512x512", type: "image/png" },
          { src: "icon-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
          { src: "icon.svg", sizes: "any", type: "image/svg+xml" },
        ],
      },
      workbox: {
        // Кэш статики; API не кэшируем как источник правды, только shell.
        navigateFallback: "/index.html",
        globPatterns: ["**/*.{js,css,html,svg,woff2}"],
      },
    }),
  ],
  server: {
    proxy: {
      "/api": {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
});
