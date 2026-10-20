<template>
  <svg class="farm-landscape" viewBox="0 0 960 740" aria-hidden="true">
    <defs>
      <linearGradient id="farm-meadow-light" x1="0" y1="0" x2=".7" y2="1"><stop stop-color="#d7dfad"/><stop offset=".45" stop-color="#bfd09a"/><stop offset="1" stop-color="#a8bf88"/></linearGradient>
      <radialGradient id="farm-morning-glow" cx=".12" cy=".08" r=".8"><stop stop-color="#fff3c2" stop-opacity=".48"/><stop offset=".5" stop-color="#fff2c9" stop-opacity=".08"/><stop offset="1" stop-color="#fff2c9" stop-opacity="0"/></radialGradient>
      <linearGradient id="farm-tree-crown" x1="0" y1="0" x2=".7" y2="1"><stop stop-color="#92b56e"/><stop offset=".5" stop-color="#769c5d"/><stop offset="1" stop-color="#527b48"/></linearGradient>
      <linearGradient id="farm-tree-lit" x1="0" y1="0" x2=".8" y2="1"><stop stop-color="#c3d994"/><stop offset=".65" stop-color="#9abc77"/><stop offset="1" stop-color="#80a466"/></linearGradient>
      <linearGradient id="farm-pond-water" x1="0" y1="0" x2=".5" y2="1"><stop stop-color="#71a99d"/><stop offset=".55" stop-color="#95c5b3"/><stop offset="1" stop-color="#bad5b6"/></linearGradient>
      <linearGradient id="farm-roof-light" x1="0" y1="0" x2=".7" y2="1"><stop stop-color="#d99b75"/><stop offset="1" stop-color="#ad7156"/></linearGradient>
      <linearGradient id="farm-wall-light" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#fff1d0"/><stop offset="1" stop-color="#e2cda5"/></linearGradient>
      <pattern id="farm-soil-grain" width="9" height="9" patternUnits="userSpaceOnUse"><circle cx="2" cy="3" r=".55" fill="#faf5cf" opacity=".25"/><circle cx="7" cy="6" r=".45" fill="#435837" opacity=".16"/></pattern>
      <pattern id="meadow-specks" width="84" height="64" patternUnits="userSpaceOnUse"><path d="m12 26 4-3m-1 7 4-2m42 20 4-3" fill="none" stroke="#90ad76" stroke-width="2" stroke-linecap="round" opacity=".35"/><ellipse cx="48" cy="14" rx="2" ry="1" fill="#e4e9bf"/></pattern>
      <symbol id="orchard-tree" viewBox="0 0 120 160"><ellipse cx="72" cy="143" rx="42" ry="17" fill="#405b39" opacity=".19" transform="rotate(15 72 143)"/><path d="m56 140 1-69h9l2 69" fill="#967756"/><path d="m60 113-16-18m18 8 16-16" stroke="#967756" stroke-width="5" stroke-linecap="round"/><path d="M14 85C0 63 17 44 30 44 26 17 59 3 79 22c29-2 43 23 32 41 17 28-7 49-36 45-21 12-56 0-61-23" fill="url(#farm-tree-crown)"/><path d="M18 69c-2-16 14-24 26-24-1-21 26-28 39-13 20-1 27 15 20 29-18 22-57 25-85 8" fill="url(#farm-tree-lit)"/><path d="M29 52q3-11 17-11m6-13q11-7 23 1" fill="none" stroke="#e0e8ac" stroke-width="3" stroke-linecap="round" opacity=".5"/><g fill="#efc57c" stroke="#b38f50" stroke-width=".6"><circle cx="34" cy="77" r="5"/><circle cx="85" cy="66" r="5"/><circle cx="64" cy="40" r="4"/></g></symbol>
      <symbol id="meadow-shrub" viewBox="0 0 100 60"><ellipse cx="50" cy="47" rx="46" ry="12" fill="#698257" opacity=".13"/><path d="M7 39C-2 22 20 13 30 22 34 0 65-1 71 21c17-9 36 15 19 24-23 13-64 12-83-6" fill="url(#farm-tree-crown)"/><path d="M15 30c5-10 16-7 23-4 6-16 25-15 30-3" fill="none" stroke="#b3c690" stroke-width="5" stroke-linecap="round"/></symbol>
    </defs>
    <rect width="960" height="740" fill="url(#farm-meadow-light)"/><rect width="960" height="740" fill="url(#meadow-specks)"/>
    <path d="M-70 575 500 290 1040 560 470 845Z" fill="#c8d7a9" opacity=".6"/>
    <!-- Every ground edge follows the same 2:1 projection as the plots. -->
    <path :d="field.road" fill="none" stroke="#b6bf8e" stroke-width="48" stroke-linejoin="round"/>
    <path :d="field.road" fill="none" stroke="#e6d8b3" stroke-width="40" stroke-linejoin="round"/>
    <path :d="field.road" fill="none" stroke="#efe3c4" stroke-width="25" stroke-linejoin="round"/>
    <path :d="field.entrance" fill="none" stroke="#e6d8b3" stroke-width="36"/><path :d="field.entrance" fill="none" stroke="#efe3c4" stroke-width="23"/>
    <g stroke-linejoin="round"><path :d="field.rails" stroke="#a38d67" stroke-width="5" transform="translate(0 3)"/><path :d="field.rails" stroke="#eadcb8" stroke-width="4"/><path v-for="(post,i) in field.fence" :key="i" :d="`M${post[0]} ${post[1]-3}v37`" stroke="#c5ac81" stroke-width="7"/><path v-for="(post,i) in field.fence" :key="`cap${i}`" :d="`M${post[0]-2} ${post[1]-6}l4 2`" stroke="#f3e7ca" stroke-width="6"/></g>
    <use href="#orchard-tree" x="610" y="79" width="120" height="160"/><use href="#orchard-tree" x="727" y="133" width="100" height="133"/><use href="#orchard-tree" x="817" y="179" width="109" height="145"/>
    <g transform="translate(340 180)" stroke-linejoin="round">
      <path :d="face([[-15,-15,0],[126,-15,0],[126,99,0],[-15,99,0]])" fill="#6c7f52" opacity=".16" transform="translate(8 8)"/>
      <path :d="face([[0,80,0],[104,80,0],[104,80,78],[0,80,78]])" fill="url(#farm-wall-light)" stroke="#c7b58c"/>
      <path :d="face([[104,0,0],[104,80,0],[104,80,78],[104,0,78]])" fill="#d9c49a" stroke="#bda77f"/>
      <path :d="face([[104,0,78],[104,40,128],[104,80,78]])" fill="#e7d6ad"/>
      <path :d="face([[-10,-10,78],[114,-10,78],[114,40,128],[-10,40,128]])" fill="#d49a79" stroke="#a46f55" stroke-width="2"/>
      <path :d="face([[-10,40,128],[114,40,128],[114,90,78],[-10,90,78]])" fill="url(#farm-roof-light)" stroke="#a46f55" stroke-width="2"/>
      <path v-for="n in 6" :key="n" :d="face([[n*19-10,40,128],[n*19-10,90,78]],false)" fill="none" stroke="#d69b76" stroke-width="2"/>
      <path :d="face([[-10,40,128],[114,40,128]],false)" fill="none" stroke="#f0c19a" stroke-width="2"/>
      <path v-for="n in 3" :key="`tile${n}`" :d="face([[-10,40+n*12,128-n*12],[114,40+n*12,128-n*12]],false)" fill="none" stroke="#8e5b45" stroke-width=".8" opacity=".3"/>
      <path :d="face([[17,80,0],[45,80,0],[45,80,51],[17,80,51]])" fill="#947359" stroke="#c4ac7e" stroke-width="3"/>
      <path :d="face([[63,80,29],[90,80,29],[90,80,56],[63,80,56]])" fill="#a9c6b7" stroke="#f9edcc" stroke-width="4"/>
      <path :d="face([[104,22,27],[104,53,27],[104,53,54],[104,22,54]])" fill="#8eafa6" stroke="#eadbb7" stroke-width="4"/>
      <path :d="face([[65,80,52],[75,80,52],[87,80,34],[77,80,34]])" fill="#e4f0d9" opacity=".55"/>
      <path :d="face([[104,25,49],[104,32,49],[104,48,32],[104,41,32]])" fill="#d9eddb" opacity=".4"/>
      <path :d="face([[0,80,77],[104,80,77]],false)" stroke="#a08b65" stroke-width="3" opacity=".3"/>
      <path :d="face([[76,80,29],[76,80,56]],false)" stroke="#f9edcc" stroke-width="2"/>
      <path :d="face([[12,80,2],[49,80,2],[49,96,2],[12,96,2]])" fill="#cbb58b"/>
      <circle cx="-42" cy="39" r="2" fill="#e6c77e"/>
    </g>
    <use href="#orchard-tree" x="104" y="139" width="120" height="160"/>
    <use href="#meadow-shrub" x="188" y="266" width="88" height="53"/>
    <g transform="translate(124 462)"><path d="M-75-3c-5-31 39-45 76-42 40-2 88 13 80 42C74 28 34 43-7 38-51 40-77 20-75-3" fill="#b2bc91"/><path d="M-67-4c-4-25 37-35 69-33C37-39 77-22 73-3 67 21 31 34-5 30-42 32-67 15-67-4" fill="url(#farm-pond-water)"/><path d="m-42-7 28-14m23 38 26-13m-31-11 17-8" stroke="#cce0ca" stroke-width="3" stroke-linecap="round"/><ellipse cx="-32" cy="15" rx="14" ry="7" fill="#739d77"/><path d="m-78 7-3-42m8 37 5-29m123-6 2-27" stroke="#7e965c" stroke-width="3"/><path d="m-81-31v-13m138-13v-12" stroke="#9c8251" stroke-width="5" stroke-linecap="round"/></g>
    <g fill="none" stroke-linecap="round"><path d="M65 449q32-20 68-11" stroke="#def0ce" stroke-width="2" opacity=".7"/><path d="M124 481q22 0 39-11" stroke="#e9f1ce" stroke-width="1.5" opacity=".8"/><ellipse cx="139" cy="450" rx="9" ry="3" stroke="#d9ecd0" opacity=".65"/><path d="m114 451 6-3m-3-2v7" stroke="#f9f5d6" stroke-width="1.5"/></g>
    <g fill="#ddd8b4"><ellipse cx="165" cy="491" rx="13" ry="6.5"/><ellipse cx="103" cy="503" rx="9" ry="4.5"/><ellipse cx="61" cy="467" rx="8" ry="4"/></g>
    <use href="#meadow-shrub" x="202" y="548" width="94" height="56"/><use href="#meadow-shrub" x="783" y="492" width="110" height="66"/>
    <g transform="translate(743 574)"><ellipse cy="16" rx="36" ry="18" fill="#6c7f52" opacity=".12"/><path d="m-29-3 25-12 29 14-25 13Z" fill="#ddbd86"/><path d="m-29-3 4 20 25 12V12Zm29 15 25-13-3 20L0 29Z" fill="#b89463"/><path d="m-17 6 2 16m12-10v15m13-17-1 15m11-20-2 16" stroke="#d5b581" stroke-width="2"/><g fill="#d7ad65" stroke="#b58a4e"><circle cx="-9" cy="-9" r="8"/><circle cx="7" cy="-6" r="9"/><circle cx="-1" cy="-16" r="8"/></g><path d="m-1-23 5-5" stroke="#82965d" stroke-width="3"/></g>
    <g transform="translate(391 640)"><ellipse cy="23" rx="35" ry="17.5" fill="#6c7f52" opacity=".13"/><path d="M0 23v-54" stroke="#ae8e64" stroke-width="6"/><path d="m-37-39 64 8v26l-64-8Z" fill="#e7ce9b" stroke="#b7986a" stroke-width="2"/><path d="m-18-25 27 3m-27 5 18 2" stroke="#9a875d" stroke-width="3" stroke-linecap="round"/></g>
    <g v-for="(flower,i) in flowers" :key="i" :transform="`translate(${flower[0]} ${flower[1]})`"><path d="M0 0v-9m0 6-4-2" stroke="#8da06a" stroke-width="2"/><circle cy="-11" r="3.5" :fill="i%2?'#f1e2b5':'#e6b5a0'"/><circle cy="-11" r="1" fill="#bd9b5c"/></g>
    <g fill="#f2e6be" opacity=".42"><ellipse cx="161" cy="369" rx="16" ry="7"/><ellipse cx="191" cy="352" rx="6" ry="3"/><ellipse cx="429" cy="191" rx="15" ry="7"/><ellipse cx="452" cy="202" rx="7" ry="3"/></g>
    <g fill="none" stroke-linecap="round" opacity=".6"><path v-for="(tuft,i) in tufts" :key="`grass${i}`" :d="`M${tuft[0]} ${tuft[1]}l-2-7m2 7 4-5`" stroke="#88a16a" stroke-width="1.6"/></g>
    <rect width="960" height="740" fill="url(#farm-soil-grain)" pointer-events="none"/>
    <rect width="960" height="740" fill="url(#farm-morning-glow)" pointer-events="none"/>
    <use href="#orchard-tree" x="-38" y="578" width="145" height="193"/><use href="#orchard-tree" x="858" y="566" width="145" height="193"/>
  </svg>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { farmGeometry } from './farmGeometry'
const props = withDefaults(defineProps<{ plots?: number }>(), { plots: 12 })
const field = computed(() => farmGeometry(props.plots))
// u/v are ground axes; z is height. Shared with the field's 2:1 diamond.
const face = (points: number[][], close = true) => points.map(([u,v,z],i) => `${i?'L':'M'}${u-v} ${(u+v)/2-z}`).join(' ') + (close?'Z':'')
const tufts = [[112,355],[146,331],[209,402],[749,590],[803,638],[694,665],[328,589],[246,614],[581,226],[748,313],[853,451],[74,554],[274,159],[472,160],[526,687]]
const flowers = [[188,553],[201,560],[179,563],[827,574],[847,584],[836,592],[557,665],[575,670],[566,678],[74,330],[88,337]]
</script>
<style scoped>
.farm-landscape{position:absolute;inset:0;width:100%;height:100%;pointer-events:none;user-select:none}
</style>
