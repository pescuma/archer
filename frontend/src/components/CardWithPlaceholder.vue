<script setup>
import { reactive } from 'vue'
import axios from 'axios'

const LOADING = 0
const OK = 1
const ERROR = 2

const props = defineProps({
  type: String,
})

const data = reactive({
  state: LOADING,
  error: '',
})

function request(urls, cb) {
  let explode = false
  if (!Array.isArray(urls)) {
    urls = [urls]
    explode = true
  }

  let ps = urls.map(function (url) {
    return axios.get(url)
  })

  Promise.all(ps)
    .then((response) => {
      try {
        let ds = response.map(function (i) {
          return i.data
        })
        if (explode) ds = ds[0]
        cb(ds)
        data.state = OK
      } catch (e) {
        console.log('Error parsing result', e)
        data.error = 'Error parsing result'
        data.state = ERROR
      }
    })
    .catch((error) => {
      data.error = error.message
      data.state = ERROR
    })
}

defineExpose({ request })
</script>

<template>
  <div class="card card-sm">
    <div class="card-body" v-if="data.state === OK">
      <slot></slot>
    </div>

    <div class="card-body placeholder-glow" v-else>
      <div class="row" v-if="props.type === 'count' && data.state === LOADING">
        <div class="col-auto">
          <span class="avatar placeholder"></span>
        </div>
        <div class="col">
          <div class="placeholder placeholder-xs col-9"></div>
          <div class="placeholder placeholder-xs col-7"></div>
        </div>
      </div>
      <div class="row" v-if="props.type === 'count' && data.state === ERROR">
        <div class="col-auto">
          <span class="bg-red-lt avatar"><i class="icon ti ti-alert-triangle"></i></span>
        </div>
        <div class="col">
          <div class="font-weight-medium text-red">{{ data.error }}</div>
        </div>
      </div>

      <div class="row" v-if="props.type === 'chart' && data.state === LOADING">
        <div class="col">
          <h3 class="card-title placeholder placeholder-xs col-7"></h3>
          <div class="chart-lg placeholder placeholder-xs col-12"></div>
        </div>
      </div>
      <div class="row" v-if="props.type === 'chart' && data.state === ERROR">
        <div class="col-auto">
          <span class="bg-red-lt avatar"><i class="icon ti ti-alert-triangle"></i></span>
        </div>
        <div class="col">
          <h3 class="cart-title font-weight-medium text-red">{{ data.error }}</h3>
          <div class="chart-lg"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
