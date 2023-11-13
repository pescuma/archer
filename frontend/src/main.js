import '@tabler/icons-webfont/tabler-icons.min.css'
import '@tabler/core/dist/css/tabler.min.css'
import '@tabler/core/dist/css/tabler-flags.min.css'
import '@tabler/core/dist/css/tabler-payments.min.css'
import '@tabler/core/dist/css/tabler-vendors.min.css'
import '@tabler/core/dist/js/tabler.min'
import { createApp } from 'vue'
import VueApexCharts from 'vue3-apexcharts'
import App from './App.vue'
import router from './router'

const app = createApp(App)

app.use(router)
app.use(VueApexCharts)

app.mount('#app')
