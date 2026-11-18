import { chromium } from 'playwright';
const BASE='http://127.0.0.1:8082';
const b=await chromium.launch({channel:'chrome',headless:true,args:['--no-sandbox']});
const p=await b.newPage({viewport:{width:1500,height:1000}});
const errs=[];
p.on('console',m=>{if(m.type()==='error')errs.push(m.text().slice(0,160))});
p.on('pageerror',e=>errs.push(String(e?.message||e).slice(0,220)));
await p.goto(BASE,{waitUntil:'load',timeout:45000}); await p.waitForTimeout(2500);
const inp=await p.locator('#app input').all(); await inp[0].fill('admin'); await inp[1].fill('admin');
await p.locator('button:has-text("登录")').first().click(); await p.waitForTimeout(5000);
await p.evaluate(()=>{window.location.hash='#/fcl/warehouse/assetWarehouse'}); await p.waitForTimeout(2000);
await p.locator('.aw-tab').nth(1).click(); await p.waitForTimeout(1000);
// set date to 2026-11-20 so existing active session (2026-11-17) doesn't block
await p.locator('.so-filter-item input[type=date]').fill('2026-11-20');
await p.locator('.so-filter .aw-btn-accent').click(); await p.waitForTimeout(1800);
let t=(await p.locator('main').innerText()).replace(/\s+/g,' ');
console.log('session started (list visible):', t.includes('Belum Scan ('));
// scan unknown
await p.locator('.so-manual input').fill('CEK-SELESAI-1');
await p.locator('.so-manual button').click(); await p.waitForTimeout(1500);
// finish
await p.locator('.so-laporan').click(); await p.waitForTimeout(2000);
t=(await p.locator('main').innerText()).replace(/\s+/g,' ');
console.log('on hasil tab:', t.includes('Riwayat Hasil Stock Opname'));
console.log('new session date 2026-11-20 present:', t.includes('2026-11-20'));
console.log('errors:', errs.length);
errs.slice(0,5).forEach(e=>console.log('ERR:',e));
await p.screenshot({path:'/tmp/finish_check.png'});
await b.close();
