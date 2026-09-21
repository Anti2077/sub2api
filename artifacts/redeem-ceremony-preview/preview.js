import { createApp, h } from 'vue'
import { createI18n } from 'vue-i18n'
import RedeemView from '@/views/user/RedeemView.vue'
import zh from '@/i18n/locales/zh'
import '@/style.css'
const app=createApp({render:()=>h('div',[
 h('div',{class:'mx-auto max-w-2xl px-6 pt-4 text-sm text-gray-600 dark:text-gray-200'},[
 '隔离预览 · 模拟 API，不使用真实兑换码 ',
 h('button',{class:'btn btn-secondary ml-2',onClick:()=>document.documentElement.classList.toggle('dark')},'切换深浅色'),
 h('p',{class:'mt-2'},'普通代码：余额 +20；CONCURRENCY：并发；SUBSCRIPTION：订阅；FAIL：失败')
 ]),h(RedeemView)
])})
app.use(createI18n({legacy:false,locale:'zh',messages:{zh}}));app.mount('#app')
