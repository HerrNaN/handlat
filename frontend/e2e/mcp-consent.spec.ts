import { expect, test } from "@playwright/test";

const request = "AAAAAAAAAAAAAAAAAAAAAAAAAA";

test("shows where Handlat will return consent before asking for a passkey", async ({ page }) => {
  await page.route(`**/api/mcp/approvals/${request}`, (route) =>
    route.fulfill({
      json: {
        name: "Example agent",
        domain: "client.example",
        callbackHost: "callback.example",
        localCallback: false,
        renewable: true,
      },
    }),
  );
  await page.goto(`/connect?request=${request}`);

  await expect(page.getByText("client.example", { exact: true })).toBeVisible();
  await expect(
    page.getByText("Handlat will send your response to callback.example."),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Approve with passkey" })).toBeVisible();
});

test("explains loopback callbacks and expiring access", async ({ page }) => {
  await page.route(`**/api/mcp/approvals/${request}`, (route) =>
    route.fulfill({
      json: {
        name: "Example agent",
        domain: "client.example",
        callbackHost: "127.0.0.1:51005",
        localCallback: true,
        renewable: false,
      },
    }),
  );
  await page.goto(`/connect?request=${request}`);

  await expect(
    page.getByText("A local application will receive your response at 127.0.0.1:51005."),
  ).toBeVisible();
  await expect(page.getByText("Access expires after an hour.", { exact: false })).toBeVisible();
});
