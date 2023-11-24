import cytoscape from 'cytoscape'
import dagre from 'cytoscape-dagre'
import nodeHtmlLabel from 'cytoscape-node-html-label'
import '@tabler/core/dist/css/tabler.min.css'
import '@tabler/core/dist/css/tabler-flags.min.css'
import '@tabler/core/dist/css/tabler-payments.min.css'
import '@tabler/core/dist/css/tabler-vendors.min.css'
import '@tabler/core/dist/js/tabler.min'
import '@tabler/icons-webfont/tabler-icons.min.css'
import * as tabler_icons from '@tabler/icons-vue'
import { createApp } from 'vue'
import VueApexCharts from 'vue3-apexcharts'
import App from './App.vue'
import router from './router'

cytoscape.use(dagre)
cytoscape.use(nodeHtmlLabel)

const app = createApp(App)

app.use(router)
app.use(VueApexCharts)

for (let c in tabler_icons) {
  app.component(c, tabler_icons[c])
}

app.mount('#app')
