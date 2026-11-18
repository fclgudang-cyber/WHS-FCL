import { chromium } from 'playwright';
const BASE = 'http://127.0.0.1:8082';
const browser = await chromium.launch({ channel: 'chrome', headless: true, args: ['--no-sandbox'] });
const page = await browser.newPage({ viewport: { width: 1600, height: 900 } });
await page.goto(BASE, { waitUntil: 'networkidle', timeout: 45000 });
await page.waitForTimeout(2000);
const loginInput = page.locator('#app input').first();
if (await loginInput.isVisible().catch(() => false)) {
  const inputs = await page.locator('#app input').all();
  await inputs[0].fill('admin');
  await inputs[1].fill('admin');
  await page.locator('button:has-text("登录"), button:has-text("Login")').first().click();
  await page.waitForTimeout(4000);
}
for (const path of ['/#/fcl/warehouse/stock', '/#/fcl/sistem/users']) {
  await page.goto(BASE + path, { waitUntil: 'networkidle' });
  await page.waitForTimeout(2500);
  const innerText = (await page.locator('#app').innerText()).split('\n').map(s => s.trim()).filter(Boolean);
  console.log('PAGE', path, 'lines:', innerText.length);
  console.log(innerText.join(' | '));
  const content = await page.locator('.ant-layout-content, main').first().innerHTML().catch(() => 'NOMAIN');
  console.log('CONTENT-HTML (first 1500):', content.slice(0, 1500).replace(/\s+/g, ' '));
  console.log('---\n');
}
await browser.close();
