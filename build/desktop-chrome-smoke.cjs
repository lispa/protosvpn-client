// Browser validates native layout and event wiring, not OS window-manager behaviour.
const {chromium}=require('/home/hermes/kube-deploy/node_modules/playwright');
const assert=require('node:assert/strict');
(async()=>{const browser=await chromium.launch({args:['--no-sandbox']});try{
 const page=await browser.newPage({viewport:{width:940,height:820}});
 await page.addInitScript(()=>{window.nativeCalls=[];window.runtime={WindowMinimise:()=>window.nativeCalls.push('min'),WindowToggleMaximise:()=>window.nativeCalls.push('max'),Quit:()=>window.nativeCalls.push('quit')};localStorage.setItem('protosvpn.language','ru');});
 await page.goto('http://127.0.0.1:18761');await page.getByRole('heading',{name:'Нет профилей',exact:true}).waitFor();
 for(const [width,height] of [[940,820],[640,640],[1440,1000]]){
  await page.setViewportSize({width,height});
  const layout=await page.evaluate(()=>{const s=document.querySelector('.app-shell').getBoundingClientRect();const title=document.querySelector('.titlebar');const button=document.querySelector('.window-controls button');return {x:s.x,y:s.y,w:s.width,h:s.height,scrollWidth:document.documentElement.scrollWidth,drag:getComputedStyle(title).getPropertyValue('--wails-draggable').trim(),noDrag:getComputedStyle(button.firstElementChild).getPropertyValue('--wails-draggable').trim()};});
  assert.equal(layout.x,0);assert.equal(layout.y,0);assert.equal(layout.w,width);assert.equal(layout.h,height);assert.equal(layout.scrollWidth,width);assert.equal(layout.drag,'drag');assert.equal(layout.noDrag,'no-drag');
  await page.getByRole('button',{name:'Свернуть окно',exact:true}).click();await page.getByRole('button',{name:'Развернуть / восстановить окно',exact:true}).click();
 }
 await page.getByRole('button',{name:'Закрыть приложение',exact:true}).click();assert.deepEqual(await page.evaluate(()=>window.nativeCalls),['min','max','min','max','min','max','quit']);
 await page.setViewportSize({width:940,height:820});await page.getByRole('button',{name:'Настройки',exact:true}).click();await page.getByRole('dialog').waitFor();await page.keyboard.press('Escape');await page.getByRole('dialog').waitFor({state:'detached'});
 console.log('PASS native-layout browser check: full-window shell at 640/940/1440px, no outer gap/overflow, drag/no-drag CSS, runtime control wiring, modal Escape');
}finally{await browser.close();}})().catch(e=>{console.error(e);process.exitCode=1;});
