import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const backendProxyTarget = process.env.VITE_PROXY_BACKEND_URL ?? "http://127.0.0.1:3000";
const gamesWebProxyTarget = process.env.VITE_PROXY_GAMES_WEB_URL ?? "http://127.0.0.1:5180";

export default defineConfig({
  plugins: [react()],
  server: {
    allowedHosts: true,
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api": {
        target: backendProxyTarget,
        changeOrigin: true
      },
      "/socket.io": {
        target: backendProxyTarget,
        changeOrigin: true,
        ws: true
      },
      "/games": {
        target: gamesWebProxyTarget,
        changeOrigin: true,
        ws: true
      }
    }
  }
});
