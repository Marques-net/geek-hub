import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "@playwright/test";

const DEFAULT_ORIGINS = [
  "http://192.168.0.100:30401",
  "http://chess.local",
  "http://localhost:5173",
  "http://localhost:8080"
];

const CLIENT_ID_PATTERN = /[0-9A-Za-z-]+\.apps\.googleusercontent\.com/;
const currentDir = path.dirname(fileURLToPath(import.meta.url));
const e2eRoot = path.resolve(currentDir, "..");
const repoRoot = path.resolve(e2eRoot, "..");
const envFilePath =
  process.env.GOOGLE_CLIENT_ENV_FILE?.trim() || path.join(repoRoot, "frontend", ".env.local");
const userDataDir = path.join(e2eRoot, ".playwright", "google-auth");

const origins = (process.env.GOOGLE_AUTH_ORIGINS ?? "")
  .split(",")
  .map((value) => value.trim())
  .filter(Boolean);

const authorizedOrigins = origins.length > 0 ? origins : DEFAULT_ORIGINS;
const clientName = process.env.GOOGLE_OAUTH_CLIENT_NAME?.trim() || "claro-games-portal-local";
const timeoutMs = Number(process.env.GOOGLE_CLIENT_ID_TIMEOUT_MS ?? 15 * 60 * 1000);
const headless = process.env.HEADLESS === "1";

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

const upsertEnvVar = (content, key, value) => {
  const line = `${key}=${value}`;
  const matcher = new RegExp(`^${key}=.*$`, "m");
  if (matcher.test(content)) {
    return content.replace(matcher, line);
  }

  const suffix = content.length > 0 && !content.endsWith("\n") ? "\n" : "";
  return `${content}${suffix}${line}\n`;
};

const readPotentialClientId = async (context) => {
  for (const page of context.pages()) {
    try {
      const bodyText = await page.locator("body").textContent({ timeout: 1_000 });
      const match = bodyText?.match(CLIENT_ID_PATTERN);
      if (match) {
        return match[0];
      }
    } catch {
      // noop
    }

    try {
      const content = await page.content();
      const match = content.match(CLIENT_ID_PATTERN);
      if (match) {
        return match[0];
      }
    } catch {
      // noop
    }
  }

  return null;
};

const printInstructions = () => {
  console.log("");
  console.log("Playwright helper para Google Client ID");
  console.log("");
  console.log("1. Faça login na sua conta Google Cloud no navegador aberto.");
  console.log("2. Acesse Google Auth Platform > Clients > Create client.");
  console.log("3. Crie um client do tipo Web application.");
  console.log(`4. Use o nome: ${clientName}`);
  console.log("5. Cadastre estas Authorized JavaScript origins:");
  for (const origin of authorizedOrigins) {
    console.log(`   - ${origin}`);
  }
  console.log("6. Quando o Client ID aparecer na tela, o script tenta capturar e gravar em frontend/.env.local.");
  console.log("");
};

const main = async () => {
  fs.mkdirSync(path.dirname(envFilePath), { recursive: true });
  fs.mkdirSync(userDataDir, { recursive: true });

  printInstructions();

  let clientId = process.env.GOOGLE_CLIENT_ID?.trim() || null;
  if (!clientId) {
    const context = await chromium.launchPersistentContext(userDataDir, {
      channel: "chrome",
      headless,
      viewport: { width: 1440, height: 960 }
    });

    const docsPage = await context.newPage();
    await docsPage.goto("https://support.google.com/cloud/answer/15544987?hl=en", {
      waitUntil: "domcontentloaded"
    });

    const consolePage = await context.newPage();
    await consolePage.goto("https://console.cloud.google.com/auth/clients", {
      waitUntil: "domcontentloaded"
    });

    const deadline = Date.now() + timeoutMs;
    while (!clientId && Date.now() < deadline) {
      clientId = await readPotentialClientId(context);
      if (clientId) {
        break;
      }

      await sleep(2_000);
    }

    await context.close();
  }

  if (!clientId) {
    console.error("Nao foi possivel capturar o Client ID automaticamente.");
    console.error("Crie o client manualmente e rode novamente com GOOGLE_CLIENT_ID=<valor>.");
    process.exitCode = 1;
    return;
  }

  const currentEnv = fs.existsSync(envFilePath) ? fs.readFileSync(envFilePath, "utf8") : "";
  const nextEnv = upsertEnvVar(currentEnv, "VITE_GOOGLE_CLIENT_ID", clientId);
  fs.writeFileSync(envFilePath, nextEnv, "utf8");

  console.log("");
  console.log(`Client ID capturado: ${clientId}`);
  console.log(`Arquivo atualizado: ${envFilePath}`);
  console.log("Agora rode o portal com esse .env.local ou replique o valor no deployment do cluster.");
};

await main();
