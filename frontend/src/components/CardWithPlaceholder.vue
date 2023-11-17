<script setup>
import { reactive } from 'vue'

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

function startLoading() {
  data.state = LOADING
  data.error = ''
}

function stopLoading(errorMessage, error) {
  if (!errorMessage) {
    data.state = OK
    data.error = ''
  } else {
    data.state = ERROR
    data.error = errorMessage
    console.log(errorMessage, error)
  }
}

async function loading(f) {
  startLoading()
  try {
    let result = await f()
    stopLoading()
    return result
  } catch (e) {
    stopLoading(e)
    throw e
  }
}

function request(urls, cb) {
  startLoading()

  let explode = false
  if (!Array.isArray(urls)) {
    urls = [urls]
    explode = true
  }

  Promise.all(urls.map(window.api.get))
    .then((responses) => {
      try {
        if (explode) responses = responses[0]

        cb(responses)
        stopLoading()
      } catch (e) {
        stopLoading('Error parsing result', e)
      }
    })
    .catch((error) => {
      stopLoading(error)
    })
}

defineExpose({ startLoading, stopLoading, loading, request })
</script>

<template>
  <div class="card card-sm" v-if="data.state === OK">
    <slot></slot>
  </div>
  <div class="card card-sm placeholder-glow" v-else>
    <div class="card-body" v-if="props.type === 'count' && data.state === LOADING">
      <div class="row">
        <div class="col-auto">
          <span class="avatar placeholder"></span>
        </div>
        <div class="col">
          <div class="placeholder placeholder-xs col-9"></div>
          <div class="placeholder placeholder-xs col-7"></div>
        </div>
      </div>
    </div>
    <div class="card-body" v-if="props.type === 'count' && data.state === ERROR">
      <div class="row">
        <div class="col-auto">
          <span class="bg-red-lt avatar"><icon-alert-triangle class="icon" /></span>
        </div>
        <div class="col">
          <div class="text-red">{{ data.error }}</div>
        </div>
      </div>
    </div>

    <div v-if="props.type === 'chart' && data.state === LOADING">
      <div class="card-header">
        <h3 class="card-title placeholder placeholder-xs col-5"></h3>
      </div>
      <div class="card-body">
        <div class="chart-lg placeholder placeholder-xs col-12"></div>
      </div>
    </div>
    <div v-if="props.type === 'chart' && data.state === ERROR">
      <div class="card-header">
        <h3 class="card-title text-red">
          <icon-alert-triangle class="icon" />
          {{ data.error }}
        </h3>
      </div>
      <div class="card-body">
        <div class="chart-lg"></div>
      </div>
    </div>

    <div v-if="props.type === 'table' && data.state === LOADING">
      <div class="card-header">
        <h3 class="card-title placeholder col-5"></h3>
      </div>
      <div class="card-body">
        <div class="placeholder col-12" style="height: 520px"></div>
      </div>
    </div>
    <div v-if="props.type === 'table' && data.state === ERROR">
      <div class="card-header">
        <h3 class="card-title text-red">
          <icon-alert-triangle class="icon" />
          {{ data.error }}
        </h3>
      </div>
      <div class="card-body">
        <div style="height: 520px"></div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
