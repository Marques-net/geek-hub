import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const backendProxyTarget = process.env.VITE_PROXY_BACKEND_URL ?? "http://127.0.0.1:3000";
const loginProxyTarget = process.env.VITE_PROXY_LOGIN_BACKEND_URL ?? "http://127.0.0.1:3001";

export default defineConfig({
  plugins: [react()],
  base: "/games/",
  server: {
    allowedHosts: true,
    host: "0.0.0.0",
    port: 5180,
    proxy: {
      "/api/auth": {
        target: loginProxyTarget,
        changeOrigin: true
      },
      "/api": {
        target: backendProxyTarget,
        changeOrigin: true
      },
      "/socket.io": {
        target: backendProxyTarget,
        ws: true,
        changeOrigin: true
      }
    }
  }
});
