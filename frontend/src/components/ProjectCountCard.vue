<script setup>
import CountCard from '@/components/CountCard.vue'

function onReceived(response, props) {
  props.text = (response.total - response.external).toLocaleString() + ' Projects'
  props.details = response.external.toLocaleString() + ' external'
}
</script>

<template>
  <CountCard url="/api/stats/count/projects" @received="onReceived" color="azure" icon="folders" />
</template>

<style scoped></style>
