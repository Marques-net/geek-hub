import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({
    plugins: [react()],
    base: "/games/",
    server: {
        allowedHosts: true,
        host: "0.0.0.0",
        port: 5180,
        proxy: {
            "/api": {
                target: "http://127.0.0.1:3000",
                changeOrigin: true
            },
            "/socket.io": {
                target: "ws://127.0.0.1:3000",
                ws: true,
                changeOrigin: true
            }
        }
    }
});
