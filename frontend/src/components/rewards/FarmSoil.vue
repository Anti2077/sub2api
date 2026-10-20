<template>
  <svg class="pod-planter" viewBox="0 0 200 100" aria-hidden="true">
    <defs>
      <linearGradient :id="`${id}-earth`" x1="0" y1="0" x2=".8" y2="1"><stop stop-color="#b49160"/><stop offset=".5" stop-color="#997048"/><stop offset="1" stop-color="#825c3c"/></linearGradient>
      <linearGradient :id="`${id}-rim`" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#dfbd81"/><stop offset=".5" stop-color="#bc945d"/><stop offset="1" stop-color="#a67b4e"/></linearGradient>
    </defs>
    <path d="m3 52 97-48 98 49-97 47Z" fill="#546440" opacity=".25"/>
    <path d="m4 48 96-48 96 48v4l-96 48L4 52Z" fill="#735136"/>
    <path d="m4 48 96-48 96 48-96 48Z" :fill="`url(#${id}-rim)`"/>
    <path d="m12 48 88-44 88 44-88 44Z" class="field-soil" :fill="`url(#${id}-earth)`"/>
    <path d="m12 48 88-44 88 44" fill="none" stroke="#705132" stroke-width="1.5" opacity=".65"/>
    <g fill="none" stroke-linecap="round">
      <path d="m32 49 68 34m-49-43 68 34m-49-43 68 34m-49-43 68 34m-49-43 68 34" stroke="#60442d" stroke-width="5" opacity=".55"/>
      <path d="m32 46 68 34m-49-43 68 34m-49-43 68 34m-49-43 68 34m-49-43 68 34" stroke="#c39a63" stroke-width="2.5"/>
      <path d="m33 45 65 32m-46-41 65 32m-46-41 65 32m-46-41 65 32m-46-41 65 32" stroke="#dfbc80" stroke-width=".6" opacity=".8"/>
    </g>
    <path d="m5 48 95-47 95 47M6 50l94 47" fill="none" stroke="#f0d39b" stroke-width="1" opacity=".6"/>
    <g fill="#e2c18b" opacity=".8"><ellipse cx="29" cy="47" rx="1.8" ry=".9"/><ellipse cx="99" cy="15" rx="2" ry="1"/><ellipse cx="167" cy="50" rx="2" ry="1"/><ellipse cx="77" cy="72" rx="1.5" ry=".8"/></g>
    <g fill="#64492e" opacity=".4"><ellipse v-for="(p,i) in grains" :key="i" :cx="p[0]" :cy="p[1]" rx=".9" ry=".5"/></g>
  </svg>
</template>
<script setup lang="ts">
import { useId } from 'vue'
const id = `soil-${useId()}`
const grains = [[50,48],[84,24],[116,33],[144,48],[94,73],[70,55],[105,55],[126,61],[105,82],[58,39],[138,43],[103,21]]
</script>
