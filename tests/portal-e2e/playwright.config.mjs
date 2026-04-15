import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? "http://127.0.0.1:4173";
const useExistingServer = process.env.PLAYWRIGHT_USE_EXISTING_SERVER === "1";
const browserChannel = process.env.PLAYWRIGHT_CHANNEL?.trim() || "chrome";

export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  expect: {
    timeout: 5_000
  },
  fullyParallel: true,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL,
    channel: browserChannel,
    headless: true,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    launchOptions: {
      args: ["--disable-crash-reporter", "--disable-crashpad"]
    }
  },
  webServer: useExistingServer
    ? undefined
    : {
        command: "npm run dev -- --host 127.0.0.1 --port 4173",
        cwd: "../frontend",
        port: 4173,
        reuseExistingServer: true,
        timeout: 120_000
      },
  projects: [
    {
      name: "portal-chrome",
      use: {
        ...devices["Desktop Chrome"]
      }
    }
  ]
});
