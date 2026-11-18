import { chromium } from 'playwright';

const BASE = 'http://127.0.0.1:8082';
const errors = [];
const logs = [];

const browser = await chromium.launch({
  channel: 'chrome',
  headless: true,
  args: ['--no-sandbox'],
});
const page = await browser.newPage({ viewport: { width: 1600, height: 900 } });
page.on('console', (m) => {
  const t = m.type();
  const text = m.text();
  if (t === 'error') errors.push(text);
  logs.push(`[${t}] ${text}`);
});
page.on('pageerror', (e) => errors.push('PAGEERROR: ' + (e?.message || e)));

function countText(h) {
  const m = h.innerText;
  return m.trim().length;
}

await page.goto(BASE, { waitUntil: 'networkidle', timeout: 45000 });
await page.waitForTimeout(3000);

const url0 = page.url();
const html0 = await page.locator('#app').innerHTML().catch(() => '');
console.log('URL after load:', url0);
console.log('#app chars:', html0.length);

// if login form visible, do login
const loginInput = page.locator('input[type="text"], input[placeholder*="用户"], input[placeholder*="username"], input[placeholder*="账号"]').first();
const visible = await loginInput.isVisible().catch(() => false);
console.log('login form visible:', visible);
if (visible) {
  const inputs = await page.locator('#app input').all();
  console.log('inputs found:', inputs.length);
  if (inputs.length >= 2) {
    await inputs[0].fill('admin');
    await inputs[1].fill('admin');
    const btn = page.locator('button:has-text("登录"), button:has-text("Login")').first();
    const btnVisible = await btn.isVisible().catch(() => false);
    console.log('login btn visible:', btnVisible);
    if (btnVisible) {
      await btn.click();
      await page.waitForTimeout(5000);
    } else {
      await inputs[1].press('Enter');
      await page.waitForTimeout(5000);
    }
  }
}
console.log('URL after login:', page.url());

for (const path of ['/#/fcl/dashboard', '/#/fcl/warehouse/stock', '/#/fcl/keuangan/kasGudang', '/#/fcl/operasional/laporanKerja', '/#/fcl/sistem/users', '/#/fcl/operasional/approvalCenter']) {
  await page.goto(BASE + path, { waitUntil: 'networkidle' }).catch(() => {});
  await page.waitForTimeout(2500);
  const text = await page.locator('#app').innerText().catch(() => '');
  console.log('PAGE', path, '→ chars:', text.trim().length, '| snippet:', JSON.stringify(text.trim().slice(0, 80)));
}

console.log('\n=== console/page errors (', errors.length, ') ===');
errors.slice(0, 20).forEach((e) => console.log('ERR:', e));
console.log('\n=== last logs ===');
logs.slice(-25).forEach((l) => console.log(l));
await browser.close();
