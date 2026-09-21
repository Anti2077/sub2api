'use strict';
const $ = id => document.getElementById(id);
const stage = document.querySelector('.stage');
const canvas = $('energy'), ctx = canvas.getContext('2d');
const phases = [...$('phases').children];
// Rasterize the glow once and reuse its small bitmap throughout the animation.
const glowSprite=document.createElement('canvas');glowSprite.width=glowSprite.height=96;
const glowCtx=glowSprite.getContext('2d'), glowGradient=glowCtx.createRadialGradient(48,48,0,48,48,48);
glowGradient.addColorStop(0,'#ffffff');glowGradient.addColorStop(.15,'#e7eeff');glowGradient.addColorStop(.4,'#9bafff90');glowGradient.addColorStop(1,'#8c9fff00');glowCtx.fillStyle=glowGradient;glowCtx.fillRect(0,0,96,96);
const reduce = matchMedia('(prefers-reduced-motion: reduce)');
let raf = 0, running = false, start = 0, speed = 1, lastConfetti = -100, completed = false;
let geom, width, height, chars = [], charOrigins = [], activePhase=-2, lastClock=-1, arrived=false;
let samples=[], prevFrame=0;
function recordPerf(){
 const out={};
 for(const [name,lo,hi] of [['gather',.2,1.3],['transfer',1.3,2.2],['arrival',2.2,2.85],['celebration',2.85,4.9]]){
  const a=samples.filter(x=>x.t>=lo&&x.t<hi), gaps=a.map(x=>x.gap).sort((a,b)=>a-b),costs=a.map(x=>x.cost).sort((a,b)=>a-b);
  out[name]={frames:a.length,averageGapMs:+(gaps.reduce((a,b)=>a+b,0)/(a.length||1)).toFixed(2),p95GapMs:gaps[Math.floor(gaps.length*.95)],p95JsMs:costs[Math.floor(costs.length*.95)]};
 }
 document.getElementById('perf').textContent=JSON.stringify(out);
}
const clamp = x => Math.max(0,Math.min(1,x));
const ease = x => 1-Math.pow(1-clamp(x),3);
const mix = (a,b,t) => a+(b-a)*t;
function size(){
 const r=stage.getBoundingClientRect(), dpr=Math.min(devicePixelRatio||1,2);width=r.width;height=r.height;canvas.width=width*dpr;canvas.height=height*dpr;ctx.setTransform(dpr,0,0,dpr,0,0);
 const code=$('codeWrap').getBoundingClientRect(), dest=$('balance').getBoundingClientRect();
 const sx=code.left-r.left+code.width*.5, sy=code.top-r.top+8;
 const tx=dest.left-r.left+dest.width*.72, ty=dest.top-r.top+dest.height*.55;
 geom={sourceLeft:code.left-r.left,sourceWidth:code.width,sx,sy,tx,ty,c1x:sx+Math.min(22,width*.04),c1y:sy-68,c2x:tx+Math.min(100,width*.18),c2y:ty+72};
}
function point(t){const g=geom,u=1-t;return{x:u*u*u*g.sx+3*u*u*t*g.c1x+3*u*t*t*g.c2x+t*t*t*g.tx,y:u*u*u*g.sy+3*u*u*t*g.c1y+3*u*t*t*g.c2y+t*t*t*g.ty};}
function phase(n){if(n===activePhase)return;activePhase=n;phases.forEach((p,i)=>p.classList.toggle('active',i===n));document.body.dataset.phase=String(n);}
function reset(){cancelAnimationFrame(raf);running=false;completed=false;arrived=false;lastClock=-1;if(window.confetti)confetti.reset();ctx.clearRect(0,0,width,height);$('balance').textContent='128.00';$('code').style.opacity='1';$('code').disabled=false;$('codeGhost').style.display='none';$('codeGhost').style.opacity='1';$('codeGhost').style.transform='';$('rainbowBorder').style.opacity='0';$('scan').style.opacity='0';$('scanRight').style.opacity='0';$('scan').style.transform='';$('scanRight').style.transform='';$('halo').style.opacity='0';$('gain').style.opacity='0';$('balanceCard').style.transform='';$('transitText').style.opacity='1';$('transitText').textContent='让灵感继续发生';$('buttonLabel').textContent='兑换 $20.00';$('redeemButton').disabled=false;$('redeemButton').classList.remove('success');$('status').textContent='演示兑换不会消耗真实兑换码';$('status').classList.remove('success');$('slow').disabled=false;$('timeLabel').textContent='READY';phase(-1);}
function finish(){recordPerf();running=false;completed=true;ctx.clearRect(0,0,width,height);$('balance').textContent='148.00';$('code').disabled=true;$('redeemButton').disabled=false;$('buttonLabel').textContent='再看一次';$('redeemButton').classList.add('success');$('status').textContent='已到账 $20.00 · 模拟兑换成功';$('status').classList.add('success');$('halo').style.opacity=0;$('balanceCard').style.transform='';$('slow').disabled=false;$('timeLabel').textContent='COMPLETE';phase(3);}
function draw(t){
 ctx.clearRect(0,0,width,height);
 // A soft upward wash, then a separate pearl appears. No gathering motion.
 const wash=clamp(t/1.3);
 $('codeGhost').style.opacity=String(1-clamp((wash-.38)/.42));
 $('codeGhost').style.transform='';
 chars.forEach(el=>{el.style.color=wash>.18?`hsl(${230-wash*175},65%,${48+wash*24}%)`:'';});
 // Three filaments and a tapered luminous tail fly along a shared cubic path.
 const travel=clamp((t-1.30)/.90);
 // Brief anticipation, fast sweep, soft landing; the tail follows speed.
 const head=1-Math.pow(1-travel,2);

 if(t>=1.30&&t<2.32){
  const h=point(head),fade=clamp((2.32-t)/.12),appear=clamp(travel/.10);
  // One small luminous pearl with a thin iridescent rim; only a short translucent wake.
  const tailStart=Math.max(0,head-.075),tail=point(tailStart);
  const wake=ctx.createLinearGradient(tail.x,tail.y,h.x,h.y+.001);
  wake.addColorStop(0,'#b9c8ef00');wake.addColorStop(1,'#bac9ed60');
  ctx.strokeStyle=wake;ctx.lineWidth=2.5;ctx.lineCap='round';ctx.globalAlpha=fade;
  ctx.beginPath();ctx.moveTo(tail.x,tail.y);
  for(let i=1;i<=12;i++){const p=point(mix(tailStart,head,i/12));ctx.lineTo(p.x,p.y);}ctx.stroke();
  const radius=(5.5+Math.sin(travel*Math.PI)*.6)*ease(appear);
  if(radius>0){
   ctx.globalAlpha=fade*.22;ctx.drawImage(glowSprite,h.x-13,h.y-13,26,26);
   const pearl=ctx.createRadialGradient(h.x-radius*.3,h.y-radius*.4,.2,h.x,h.y,radius);
   pearl.addColorStop(0,'#ffffff');pearl.addColorStop(.45,'#fbfcff');pearl.addColorStop(.8,'#e2e8f8');pearl.addColorStop(1,'#b8c9eb');
   ctx.globalAlpha=fade;ctx.fillStyle=pearl;ctx.beginPath();ctx.arc(h.x,h.y,radius,0,Math.PI*2);ctx.fill();
   const rim=ctx.createLinearGradient(h.x-radius,h.y-radius,h.x+radius,h.y+radius);
   rim.addColorStop(0,'#91c8f7');rim.addColorStop(.38,'#b3a2ec');rim.addColorStop(.7,'#edb7d1');rim.addColorStop(1,'#f0d5bb');
   ctx.strokeStyle=rim;ctx.lineWidth=.9;ctx.stroke();
   ctx.fillStyle='#ffffff';ctx.globalAlpha=fade*.95;ctx.beginPath();ctx.ellipse(h.x-radius*.28,h.y-radius*.35,radius*.35,radius*.18,-.6,0,Math.PI*2);ctx.fill();
  }
 }
 ctx.shadowBlur=0;ctx.globalAlpha=1;
}
function tick(now){if(!running)return;const costStart=performance.now(),gap=prevFrame?now-prevFrame:0;prevFrame=now;const elapsed=(now-start)/1000/speed;
 // Compress the opening to 0.75s and the flight to 0.60s; leave time to read the result.
 const t=elapsed<.75?elapsed/ .75*1.3:elapsed<1.35?1.3+(elapsed-.75)/.60*.9:2.2+(elapsed-1.35);if(Math.floor(t*10)!==lastClock){lastClock=Math.floor(t*10);$('timeLabel').textContent=elapsed.toFixed(1)+'s';}
 phase(t<1.3?0:t<2.2?1:t<2.85?2:3);
 const sweep=clamp(t/1.3);
 $('scan').style.opacity=String(Math.sin(sweep*Math.PI)*.8);
 $('scan').style.transform=`translateY(${-sweep*230}%)`;
 $('scanRight').style.opacity='0';
 $('rainbowBorder').style.opacity='0';
 $('transitText').style.opacity=t<1?String(1-clamp(t/.4)):t>2.8?String(clamp((t-2.8)/.5)):0;
 if(t>=2.2&&t<=2.92){const p=clamp((t-2.2)/.65),a=Math.sin(p*Math.PI);const value=(128+20*ease(p)).toFixed(2);if($('balance').textContent!==value)$('balance').textContent=value;$('halo').style.opacity=String(a*.9);$('balanceCard').style.transform=`scale(${1+a*.014})`;$('gain').style.opacity=String(clamp(p*4));$('gain').style.transform=`translateY(${8-ease(p)*13}px)`;}
 if(t>=2.85&&!arrived){arrived=true;$('transitText').textContent='新的可能，已经到账';$('balance').textContent='148.00';$('balanceCard').style.transform='';$('halo').style.opacity='0';$('buttonLabel').textContent='兑换成功';$('redeemButton').classList.add('success');$('status').textContent='已到账 $20.00 · 模拟兑换成功';$('status').classList.add('success');}
 if(t>2.85&&t<4.9&&t-lastConfetti>.032&&window.confetti){lastConfetti=t;const opts={particleCount:2,spread:55,startVelocity:48,ticks:190,gravity:.9,scalar:.95,colors:['#38bdf8','#a78bfa','#f472b6','#fbbf24','#34d399'],disableForReducedMotion:true};confetti({...opts,angle:60,origin:{x:0,y:.64}});confetti({...opts,angle:120,origin:{x:1,y:.64}});}
 if(t<2.4)draw(t);samples.push({t,gap,cost:performance.now()-costStart});if(t<5.9)raf=requestAnimationFrame(tick);else finish();
}
function play(event){event.preventDefault();if(running||!$('code').value.trim())return;reset();size();$('codeGhost').replaceChildren();chars=[...$('code').value.trim()].map(c=>{const span=document.createElement('span');span.textContent=c;$('codeGhost').append(span);return span;});$('codeGhost').style.display='flex';$('code').style.opacity='0';$('code').disabled=true;$('redeemButton').disabled=true;$('slow').disabled=true;$('buttonLabel').textContent='正在兑换…';$('status').textContent='模拟验证成功 · 正在将额度存入账户';lastConfetti=-100;running=true;speed=$('slow').checked?2:1;
 const parent=stage.getBoundingClientRect();charOrigins=chars.map(el=>{const r=el.getBoundingClientRect();return{x:r.left-parent.left+r.width/2,y:r.top-parent.top+r.height/2};});
 if(reduce.matches){$('gain').style.opacity=1;finish();return;}samples=[];prevFrame=0;start=performance.now();raf=requestAnimationFrame(tick);
}
$('redeemForm').addEventListener('submit',play);$('reset').addEventListener('click',reset);window.addEventListener('resize',size);reduce.addEventListener('change',()=>{if(reduce.matches&&running){cancelAnimationFrame(raf);if(window.confetti)confetti.reset();finish();}});size();
