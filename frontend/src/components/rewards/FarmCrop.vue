<template>
  <span class="farm-crop-image" aria-hidden="true">
    <svg :viewBox="field?'0 0 200 200':'20 65 160 120'" class="farm-plant-art">
      <defs>
        <linearGradient :id="`${id}-leaf`" x1="0" y1="0" x2=".8" y2="1"><stop stop-color="#bddb8b"/><stop offset=".42" stop-color="#8db66c"/><stop offset="1" stop-color="#527944"/></linearGradient>
        <linearGradient :id="`${id}-heart`" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#e4edb5"/><stop offset=".5" stop-color="#bbd694"/><stop offset="1" stop-color="#7ca164"/></linearGradient>
        <radialGradient :id="`${id}-gold`" cx=".3" cy=".25" r=".8"><stop stop-color="#ffe6a4"/><stop offset=".55" stop-color="#e4b66a"/><stop offset="1" stop-color="#b9813e"/></radialGradient>
        <radialGradient :id="`${id}-petal`" cx=".3" cy=".2" r=".85"><stop stop-color="#ffe1d2"/><stop offset=".5" stop-color="#e7afa1"/><stop offset="1" stop-color="#bc7d79"/></radialGradient>
        <linearGradient :id="`${id}-moon`" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#e5dbf2"/><stop offset=".45" stop-color="#bfb0d6"/><stop offset="1" stop-color="#8e7aa9"/></linearGradient>
      </defs>
      <g v-for="(plant,i) in plants" :key="i" :transform="`translate(${plant[0]} ${plant[1]})`">
        <ellipse cx="5" cy="3" rx="19" ry="8" fill="#443f28" opacity=".14" transform="rotate(18)"/>
        <ellipse cy="1" rx="9" ry="4" fill="#443f28" opacity=".16"/>
        <g :transform="young?'scale(.68)':'scale(1)'" stroke-linejoin="round" stroke-linecap="round">
          <path d="M0 0v-34" fill="none" stroke="#557b47" stroke-width="3"/>
          <path d="M0-7c-16 1-24-10-21-18 11-2 21 5 21 15m0-9c15 1 22-10 19-17C8-37 0-29 0-22"  :fill="`url(#${id}-leaf)`" stroke="#6d8c59" stroke-width="1.2"/>
          <path d="M-17-21q8 7 15 9m5-12 10-7" fill="none" stroke="#dbecb0" stroke-width="1" opacity=".75"/>
          <template v-if="!young">
            <g v-if="kind==='cluster'" stroke="#c09a55" stroke-width="1.2"><path d="m0-25-11-13m11 5 11-13" stroke="#6d8c59" stroke-width="2"/><circle cx="-12" cy="-36" r="10" :fill="`url(#${id}-gold)`"/><circle cx="12" cy="-43" r="10" :fill="`url(#${id}-gold)`"/><circle cy="-27" r="11" :fill="`url(#${id}-gold)`"/><path d="m-16-40 3-2m20-5 3-1m-15 17 3-2" stroke="#f8e3ae" stroke-width="3"/></g>
            <g v-else-if="kind==='bloom'"><path d="M0-33c-19 6-28-11-15-18-9-16 10-24 17-12 13-11 27 4 17 14 15 11 0 29-13 17Z" :fill="`url(#${id}-petal)`" stroke="#bc8c82" stroke-width="1.2"/><path d="M0-43c-7-3-11-8-10-13m12 13 9-10m-11 13-7 7" stroke="#f2cdc0" stroke-width="2"/><circle cy="-44" r="7" fill="#e3c67f"/><circle cy="-44" r="3" fill="#b69b5d"/></g>
            <g v-else-if="kind==='moon'" stroke="#9084a8" stroke-width="1.2"><path d="m0-25-12-10m12 2 12-10" stroke="#6d8c59" stroke-width="2"/><path d="M-12-49c-4 7-12 11-12 20 0 12 24 12 24 0 0-9-8-13-12-20" :fill="`url(#${id}-moon)`"/><path d="M12-57C8-50 1-47 1-38c0 11 22 11 22 0 0-9-7-12-11-19" :fill="`url(#${id}-moon)`"/><path d="M0-32c-3 6-9 9-9 16 0 9 18 9 18 0 0-7-6-10-9-16" :fill="`url(#${id}-moon)`"/><path d="m-18-21 2 2m7-15 2 2m3 19 2 2" stroke="#e9e0f2" stroke-width="3"/></g>
            <g v-else stroke="#6d8c59" stroke-width="1.2"><path d="M0-8C-23-8-29-30-19-40c10-3 15 7 19 16 4-13 10-22 20-17C31-28 22-8 0-8" :fill="`url(#${id}-leaf)`"/><path d="M0-8c-14-8-16-28-5-37 13 3 19 23 5 37" :fill="`url(#${id}-heart)`"/><path d="m0-17-8-13 7-14 9 14Z" fill="#e2edb8" stroke="#a7bd88"/><path d="m0-17-1-27" stroke="#f8f5d8"/></g>
          </template>
          <ellipse cx="-11" cy="-22" rx="1.6" ry="1" fill="#f5ffdf" opacity=".8" transform="rotate(28 -11 -22)"/>
        </g>
      </g>
    </svg>
  </span>
</template>
<script setup lang="ts">
import { useId } from 'vue'
const id = `crop-${useId()}`
defineProps<{ kind: string; young?: boolean; field?: boolean }>()
const plants = [[100,129],[61,149],[139,149],[100,169]]
</script>
<style scoped>
.farm-plant-art{width:100%;height:100%;overflow:visible;display:block}
</style>
