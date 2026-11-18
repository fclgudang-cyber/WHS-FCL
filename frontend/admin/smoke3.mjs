import { chromium } from 'playwright';
const BASE = 'http://127.0.0.1:8082';
const errors = [];
const browser = await chromium.launch({ channel: 'chrome', headless: true, args: ['--no-sandbox'] });
const page = await browser.newPage({ viewport: { width: 1600, height: 900 } });
page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
page.on('pageerror', (e) => errors.push('PAGEERROR: ' + (e?.message || e)));
page.on('requestfailed', (r) => errors.push('REQFAIL: ' + r.url()));
await page.goto(BASE, { waitUntil: 'networkidle', timeout: 45000 });
await page.waitForTimeout(2500);
const li = page.locator('#app input').first();
if (await li.isVisible().catch(() => false)) {
  const inputs = await page.locator('#app input').all();
  await inputs[0].fill('admin');
  await inputs[1].fill('admin');
  await page.locator('button:has-text("登录"), button:has-text("Login")').first().click();
  await page.waitForTimeout(4000);
}
const routes = ['/analytics','/fcl/dashboard','/fcl/warehouse/stock','/fcl/warehouse/order','/fcl/warehouse/inbound','/fcl/keuangan/kasGudang','/fcl/keuangan/pettyCash','/fcl/keuangan/klaim','/fcl/operasional/laporanKerja','/fcl/operasional/instruksiLembur','/fcl/operasional/ijin','/fcl/operasional/lembur','/fcl/operasional/pengajuanAsset','/fcl/operasional/approvalDashboard','/fcl/operasional/approvalCenter','/fcl/sdm/karyawan','/fcl/sdm/jadwalShift','/fcl/sistem/users'];
for (const r of routes) {
  await page.goto(BASE + '/#' + r, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1800);
  const txt = (await page.locator('#app').innerText()).trim();
  console.log('PAGE', r, '→ chars:', txt.length, '| err-free so far:', errors.length);
}
console.log('\nerrors:', errors.length);
errors.slice(0, 15).forEach(e => console.log('ERR:', e));
await browser.close();
