var _a;
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
var backendProxyTarget = (_a = process.env.VITE_PROXY_BACKEND_URL) !== null && _a !== void 0 ? _a : "http://127.0.0.1:3000";
export default defineConfig({
    plugins: [react()],
    base: "/games/",
    server: {
        allowedHosts: true,
        host: "0.0.0.0",
        port: 5180,
        proxy: {
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
