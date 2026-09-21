'use strict';
const $=id=>document.getElementById(id);
const clamp=x=>Math.max(0,Math.min(1,x));
const ease=x=>1-Math.pow(1-clamp(x),3);
const attached=$('stub').cloneNode(true);
attached.id='attachedStub';attached.setAttribute('aria-hidden','true');attached.inert=true;attached.removeAttribute('tabindex');attached.removeAttribute('role');
attached.querySelectorAll('[id]').forEach(el=>el.removeAttribute('id'));
$('form').insertBefore(attached,$('stub'));
const reduced=matchMedia('(prefers-reduced-motion: reduce)');
let pointer=null,origin=null,progress=0,keyboardStart=null;
let raf=0,busy=false,start=0,slow=1,lastBurst=0,done=false;
function reset(){cancelAnimationFrame(raf);if(pointer!==null&&$('stub').hasPointerCapture(pointer))$('stub').releasePointerCapture(pointer);pointer=null;keyboardStart=null;progress=0;confetti.reset();busy=false;done=false;$('form').style.transform='';$('stub').style.transform='';$('stub').style.opacity='1';$('stub').style.clipPath='';attached.style.clipPath='';attached.style.opacity='0';$('stamp').style.opacity='0';$('stamp').style.transform='rotate(-12deg) scale(1.2)';$('balance').textContent='128.00';$('gain').style.opacity='0';$('gain').style.transform='translateY(6px)';$('code').disabled=false;$('redeem').disabled=false;$('slow').disabled=false;$('receiptState').textContent='等待兑换';$('stub').removeAttribute('aria-disabled');$('status').textContent='捏住右侧票根向右下方拉；未撕断就松手会回弹';}
function finish(){busy=false;done=true;$('stub').style.opacity='0';attached.style.opacity='0';$('balance').textContent='148.00';$('gain').style.opacity='1';$('gain').style.transform='translateY(0)';$('stamp').style.opacity='1';$('stamp').style.transform='rotate(-12deg) scale(1)';$('receiptState').textContent='已核销 · 额度已到账';$('status').textContent='兑换成功，$20.00 已存入账户 · 重置可再次体验';$('slow').disabled=false;$('form').style.transform='';}
function tick(now){if(!busy)return;const t=(now-start)/1000/slow;
 // The attached corner travels down the perforation; the released flap pivots around it.
 const tear=clamp((t-.14)/.55),pull=Math.sin(tear*Math.PI),fall=clamp((t-.69)/.55);
 $('form').style.transform=`translateY(${-Math.sin(clamp(t/.7)*Math.PI)*2}px)`;
 if(t<.69){
  const angle=tear*12,dx=pull*5;
  $('stub').style.transformOrigin=`0 ${tear*100}%`;
  $('stub').style.transform=`translate(${dx+tear*3}px,${tear*4}px) rotate(${angle}deg)`;
  attached.style.opacity='1';attached.style.clipPath=`polygon(0 ${tear*100}%,100% ${tear*100}%,100% 100%,0 100%)`;
  const points=[`0% ${tear*100}%`];
  for(let i=26;i>=0;i--){const y=i/26*tear*100;points.push(`${i%2?2:0}% ${y}%`)}
  points.push('100% 0%',`100% ${tear*100}%`);$('stub').style.clipPath=`polygon(${points.join(',')})`;
 }else{
  attached.style.opacity='0';
  $('stub').style.transformOrigin='0 100%';$('stub').style.transform=`translate(${12+fall*65}px,${8+fall*fall*85}px) rotate(${17+fall*16}deg)`;
  $('stub').style.opacity=String(1-ease(clamp((fall-.25)/.75)));
 }
 if(t>=.69){const p=clamp((t-.69)/.55);$('balance').textContent=(128+20*ease(p)).toFixed(2);$('gain').style.opacity=String(ease(p));$('gain').style.transform=`translateY(${6*(1-ease(p))}px)`;$('stamp').style.opacity=String(ease(clamp((t-.84)/.25)));$('stamp').style.transform=`rotate(-12deg) scale(${1.2-.2*ease(clamp((t-.84)/.25))})`;$('receiptState').textContent='已核销 · 额度已到账';$('status').textContent='兑换成功，$20.00 已存入账户';}
 if(t>1.25&&t<3.0&&now-lastBurst>32){lastBurst=now;const opts={particleCount:2,spread:55,startVelocity:46,ticks:180,gravity:.9,colors:['#38bdf8','#a78bfa','#f472b6','#fbbf24','#34d399'],disableForReducedMotion:true};confetti({...opts,angle:60,origin:{x:0,y:.65}});confetti({...opts,angle:120,origin:{x:1,y:.65}});}
 if(t<4)raf=requestAnimationFrame(tick);else finish();}
function renderTear(p){
 const stub=$('stub'),front=Math.max(.001,p)*100;
 attached.style.opacity='1';attached.style.clipPath=`polygon(0 ${front}%,100% ${front}%,100% 100%,0 100%)`;
 const edge=[`0% ${front}%`];for(let i=26;i>=0;i--)edge.push(`${i%2?2:0}% ${i/26*front}%`);
 edge.push('100% 0%',`100% ${front}%`);stub.style.clipPath=`polygon(${edge.join(',')})`;
 stub.style.transformOrigin=`0 ${front}%`;stub.style.transform=`translate(${p*12}px,${p*8}px) rotate(${p*17}deg)`;
}
function canTear(){if(busy||done)return false;if(!$('code').value.trim()){$('status').textContent='先在虚线上方填入兑换码';$('code').focus();return false;}return true;}
function completeTear(){
 if(busy||done)return;
 pointer=null;keyboardStart=null;busy=true;$('code').disabled=true;$('stub').setAttribute('aria-disabled','true');$('slow').disabled=true;
 $('status').textContent='票根已撕下 · 模拟兑换成功';slow=$('slow').checked?2:1;lastBurst=0;
 if(reduced.matches){finish();return;}
 start=performance.now()-.69*1000*slow;raf=requestAnimationFrame(tick);
}
function springBack(){const from=progress,at=performance.now();pointer=null;keyboardStart=null;
 function frame(now){const t=clamp((now-at)/240);renderTear(from*(1-ease(t)));if(t<1)raf=requestAnimationFrame(frame);else{progress=0;$('stub').style.transform='';$('stub').style.clipPath='';attached.style.opacity='0';$('code').disabled=false;}}
 raf=requestAnimationFrame(frame);$('status').textContent='还没撕断，抓住票根再往外拉一点';
}
const stub=$('stub');
stub.addEventListener('pointerdown',e=>{if(e.button!==0||pointer!==null||!canTear())return;cancelAnimationFrame(raf);pointer=e.pointerId;origin={x:e.clientX,y:e.clientY};progress=0;stub.setPointerCapture(pointer);$('code').disabled=true;});
stub.addEventListener('pointermove',e=>{if(e.pointerId!==pointer)return;const dx=Math.max(0,e.clientX-origin.x),dy=Math.max(0,e.clientY-origin.y);progress=clamp((dx*.65+dy)/120);renderTear(progress);if(progress>=1){const id=pointer;completeTear();if(stub.hasPointerCapture(id))stub.releasePointerCapture(id);}});
stub.addEventListener('pointerup',e=>{if(e.pointerId===pointer)springBack();});
stub.addEventListener('pointercancel',e=>{if(e.pointerId===pointer)springBack();});
stub.addEventListener('lostpointercapture',e=>{if(e.pointerId===pointer)springBack();});
// Keyboard equivalent requires a deliberate hold; clicking never redeems.
stub.addEventListener('keydown',e=>{if(e.key!==' '&&e.key!=='Enter')return;e.preventDefault();if(e.repeat||!canTear())return;cancelAnimationFrame(raf);keyboardStart=performance.now();$('code').disabled=true;
 const frame=now=>{if(keyboardStart===null)return;progress=clamp((now-keyboardStart)/650);renderTear(progress);$('status').textContent=progress===1?'松开按键完成撕票':'按住按键撕开票根';if(progress<1)raf=requestAnimationFrame(frame);};raf=requestAnimationFrame(frame);
});
stub.addEventListener('keyup',e=>{if((e.key===' '||e.key==='Enter')&&keyboardStart!==null){e.preventDefault();if(progress===1)completeTear();else springBack();}});
stub.addEventListener('blur',()=>{if(keyboardStart!==null)springBack();});
$('form').addEventListener('submit',e=>e.preventDefault());
$('reset').addEventListener('click',reset);
reduced.addEventListener('change',()=>{if(busy&&reduced.matches){cancelAnimationFrame(raf);confetti.reset();finish();}});
