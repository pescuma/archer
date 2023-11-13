<script setup>
import { reactive } from 'vue'
import axios from 'axios'

const LOADING = 0
const OK = 1
const ERROR = 2

const props = defineProps({
  url: String,
  color: String,
  icon: String,
  text: String,
  details: String,
})

const emit = defineEmits(['received'])

const data = reactive({
  state: LOADING,
  error: '',
  color: props.color,
  icon: props.icon,
  text: props.text,
  details: props.details,
})

axios
  .get(props.url)
  .then((response) => {
    try {
      emit('received', response.data, data)
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
</script>

<template>
  <div class="card card-sm">
    <div class="card-body">
      <div class="row align-items-center" v-if="data.state === OK">
        <div class="col-auto">
          <span :class="'bg-' + data.color + ' text-white avatar'">
            <i :class="'icon ti ti-' + data.icon"></i>
          </span>
        </div>
        <div class="col">
          <div class="font-weight-medium">{{ data.text }}</div>
          <div class="text-muted">{{ data.details }}</div>
        </div>
      </div>

      <div class="row align-items-center" v-if="data.state === LOADING">
        <div class="col-auto">
          <span class="avatar placeholder"></span>
        </div>
        <div class="col">
          <div class="placeholder placeholder-xs col-9"></div>
          <div class="placeholder placeholder-xs col-7"></div>
        </div>
      </div>

      <div class="row align-items-center" v-if="data.state === ERROR">
        <div class="col-auto">
          <span class="bg-red-lt avatar"><i class="icon ti ti-alert-triangle"></i></span>
        </div>
        <div class="col">
          <div class="font-weight-medium text-red">{{ data.error }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
