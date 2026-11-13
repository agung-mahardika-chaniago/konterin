import { test, expect, request as playwrightRequest, type APIRequestContext } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const EMAIL = process.env.E2E_EMAIL ?? 'owner@fixphone.local';
const PASSWORD = process.env.E2E_PASSWORD ?? 'Owner#12345';
const API_URL = (process.env.E2E_API_URL ?? 'http://127.0.0.1:8080/api/v1').replace(/\/$/, '');

// The UI keeps the access token in memory, so status transitions are driven
// through a separate API session instead of a fragile chain of dialogs. The
// state machine itself is enforced by the server and covered by the Go tests.
async function loginApi(): Promise<{ ctx: APIRequestContext; token: string }> {
  const ctx = await playwrightRequest.newContext();
  const response = await ctx.post(`${API_URL}/auth/login`, {
    data: { email: EMAIL, password: PASSWORD },
  });
  const text = await response.text();
  expect(response.ok(), `login API gagal (${response.status()}): ${text}`).toBeTruthy();
  const payload = JSON.parse(text) as { data: { access_token: string } };
  return { ctx, token: payload.data.access_token };
}

async function advanceStatus(
  ctx: APIRequestContext,
  token: string,
  orderID: string,
  statuses: string[],
) {
  for (const status of statuses) {
    const response = await ctx.patch(`${API_URL}/service-orders/${orderID}/status`, {
      data: { status },
      headers: { Authorization: `Bearer ${token}` },
    });
    const text = await response.text();
    expect(response.ok(), `transisi ke ${status} gagal (${response.status()}): ${text}`).toBeTruthy();
  }
}

test('alur servis lengkap: login, intake, diagnosis, invoice, bayar, ambil, tracking', async ({ page }) => {
  const phone = `0812${Date.now().toString().slice(-8)}`;

  // --- Login ---------------------------------------------------------------
  await page.goto('/login');
  await page.locator('#email').fill(EMAIL);
  await page.locator('#password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Masuk' }).click();
  await expect(page).toHaveURL(/\/(dashboard|services|invoices|cashier)/);

  // --- Intake --------------------------------------------------------------
  await page.goto('/services/new');
  await page.locator('#create-cust-name').fill('Pelanggan E2E');
  await page.locator('#create-cust-phone').fill(phone);
  await page.locator('#dev-brand').fill('Samsung');
  await page.locator('#dev-model').fill('Galaxy A54');
  await page.locator('#complaint-text').fill('Layar blank setelah perangkat jatuh');
  await page.locator('#est-cost').fill('250000');
  await page.getByRole('button', { name: 'Terbitkan Tiket Servis' }).click();

  await expect(page.getByRole('heading', { name: 'Penerimaan Servis Berhasil!' })).toBeVisible();
  const ticketText = await page.getByText(/^FP-\d{6}-\d{4}$/).first().textContent();
  const ticketNo = (ticketText ?? '').trim();
  expect(ticketNo).toMatch(/^FP-\d{6}-\d{4}$/);

  await page.getByRole('button', { name: 'Lihat Detail Servis' }).click();
  await expect(page).toHaveURL(/\/services\/[0-9a-f-]{36}$/);
  const orderID = page.url().split('/').pop() as string;

  // --- Diagnosis -----------------------------------------------------------
  await page.getByRole('button', { name: 'Isi Diagnosis' }).click();
  await page.locator('#diag-diagnosis').fill('LCD rusak akibat benturan');
  await page.locator('#diag-action').fill('Ganti LCD original');
  await page.getByRole('button', { name: 'Simpan Diagnosis' }).click();
  await expect(page.getByText('Ganti LCD original')).toBeVisible();

  // --- Service item --------------------------------------------------------
  await page.getByRole('button', { name: 'Tambah Item' }).first().click();
  const itemDialog = page.getByRole('dialog');
  await itemDialog.locator('#item-type').selectOption('service');
  await itemDialog.locator('#item-description').fill('Jasa ganti LCD');
  await itemDialog.locator('#item-price').fill('200000');
  await itemDialog.getByRole('button', { name: 'Tambah Item' }).click();
  await expect(itemDialog).toBeHidden();
  await expect(page.getByText('Jasa ganti LCD').first()).toBeVisible();

  // --- Invoice + payment ---------------------------------------------------
  await page.getByRole('link', { name: /Buka Invoice/ }).click();
  await expect(page).toHaveURL(/\/invoices\/[0-9a-f-]{36}$/);

  page.once('dialog', (dialog) => dialog.accept());
  await page.getByRole('button', { name: 'Finalisasi' }).click();
  await expect(page.getByText('Belum Dibayar').first()).toBeVisible();

  await page.getByRole('button', { name: 'Terima Pembayaran' }).first().click();
  const paymentDialog = page.getByRole('dialog');
  await paymentDialog.locator('#payment-amount').fill('200000');
  await paymentDialog.getByRole('button', { name: 'Simpan Pembayaran' }).click();
  await expect(paymentDialog).toBeHidden();
  await expect(page.getByText('Lunas').first()).toBeVisible();

  // --- Pickup --------------------------------------------------------------
  const { ctx, token } = await loginApi();
  try {
    await advanceStatus(ctx, token, orderID, [
      'waiting_diagnosis',
      'in_progress',
      'testing',
      'completed',
      'ready_for_pickup',
      'picked_up',
    ]);
  } finally {
    await ctx.dispose();
  }

  await page.goto(`/services/${orderID}`);
  await expect(page.getByText('Sudah Diambil').first()).toBeVisible();

  // --- Public tracking -----------------------------------------------------
  await page.goto('/track');
  await page.locator('#ticket').fill(ticketNo);
  await page.locator('#last4').fill(phone.slice(-4));
  await page.getByRole('button', { name: 'Lacak' }).click();
  await expect(page.getByText(ticketNo).first()).toBeVisible();
});

test('halaman login tidak memiliki pelanggaran aksesibilitas kritis', async ({ page }) => {
  await page.goto('/login');

  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();

  const critical = results.violations.filter((violation) => violation.impact === 'critical');
  expect(
    critical.map((violation) => ({
      id: violation.id,
      help: violation.help,
      nodes: violation.nodes.map((node) => node.target),
    })),
    'pelanggaran aksesibilitas kritis',
  ).toEqual([]);
});
