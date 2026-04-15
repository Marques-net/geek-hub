import { expect, test } from "@playwright/test";

test.describe("portal frontend", () => {
  test("renderiza home com login e acesso visitante", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "Entre no hub de jogos" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Continuar como visitante" })).toBeVisible();
    await expect(page.getByText("Entrar com Google")).toBeVisible();
  });

  test("abre o menu ao continuar como visitante", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Continuar como visitante" }).click();

    await expect(page.getByRole("heading", { name: "Escolha o que jogar" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Abrir xadrez" })).toBeVisible();
    await expect(page.getByText("Xadrez Online")).toBeVisible();
  });
});
