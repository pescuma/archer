import '@tabler/icons-webfont/tabler-icons.min.css'
import '@tabler/core/dist/css/tabler.min.css'
import '@tabler/core/dist/css/tabler-flags.min.css'
import '@tabler/core/dist/css/tabler-payments.min.css'
import '@tabler/core/dist/css/tabler-vendors.min.css'
import '@tabler/core/dist/js/tabler.min'
import { LRUCache } from 'lru-cache'
import axios from 'axios'
import { createApp } from 'vue'
import VueApexCharts from 'vue3-apexcharts'
import App from './App.vue'
import router from './router'

const api = {
  cache: new LRUCache({ max: 100 }),
}
api.get = async function (url) {
  await new Promise((resolve) => setTimeout(resolve, 1))

  let result = api.cache.get(url)
  if (result) {
    return result
  }

  let response = await axios.get(url)
  api.cache.set(url, response.data)
  return response.data
}
window.api = api

const app = createApp(App)

app.use(router)
app.use(VueApexCharts)

app.mount('#app')
