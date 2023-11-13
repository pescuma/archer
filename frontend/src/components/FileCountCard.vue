<script setup>
import CountCard from '@/components/CountCard.vue'

function onReceived(response, props) {
  props.text = response.total.toLocaleString() + ' Files'
  props.details = response.deleted.toLocaleString() + ' deleted'
}
</script>

<template>
  <CountCard url="/api/stats/count/files" @received="onReceived" color="blue" icon="files" />
</template>

<style scoped></style>
