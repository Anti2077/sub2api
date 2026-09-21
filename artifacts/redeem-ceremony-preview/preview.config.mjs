import { defineConfig } from '../../frontend/node_modules/vite/dist/node/index.js'
import vue from '../../frontend/node_modules/@vitejs/plugin-vue/dist/index.mjs'
import { fileURLToPath } from 'node:url'
const root=fileURLToPath(new URL('./',import.meta.url))
const frontend=fileURLToPath(new URL('../../frontend/',import.meta.url))
export default defineConfig({
 root, plugins:[vue()],
 resolve:{alias:[
  {find:/^@\/(api|stores\/(auth|app|subscriptions))$/,replacement:root+'mock.js'},
  {find:'@/components/layout/AppLayout.vue',replacement:root+'PreviewLayout.vue'},
  {find:'@',replacement:frontend+'src'},
  {find:'vue',replacement:frontend+'node_modules/vue/dist/vue.esm-bundler.js'},
  {find:'vue-i18n',replacement:frontend+'node_modules/vue-i18n/dist/vue-i18n.esm-bundler.js'}
 ]},
 css:{postcss:frontend},
 server:{host:'127.0.0.1',port:4180,strictPort:true,fs:{allow:[frontend,root]}},
 define:{__VUE_I18N_FULL_INSTALL__:true,__VUE_I18N_LEGACY_API__:false,__INTLIFY_PROD_DEVTOOLS__:false}
})
