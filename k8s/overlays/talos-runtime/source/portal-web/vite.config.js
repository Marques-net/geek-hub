import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({
    plugins: [react()],
    server: {
        allowedHosts: true,
        host: "0.0.0.0",
        port: 5173,
        proxy: {
            "/api": {
                target: "http://127.0.0.1:3000",
                changeOrigin: true
            },
            "/socket.io": {
                target: "ws://127.0.0.1:3000",
                changeOrigin: true,
                ws: true
            },
            "/chess": {
                target: "http://127.0.0.1:5180",
                changeOrigin: true,
                ws: true
            }
        }
    }
});
