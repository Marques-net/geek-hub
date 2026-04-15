var _a, _b;
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
var backendProxyTarget = (_a = process.env.VITE_PROXY_BACKEND_URL) !== null && _a !== void 0 ? _a : "http://127.0.0.1:3000";
var gamesWebProxyTarget = (_b = process.env.VITE_PROXY_GAMES_WEB_URL) !== null && _b !== void 0 ? _b : "http://127.0.0.1:5180";
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
