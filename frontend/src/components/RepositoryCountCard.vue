<script setup>
import CountCard from '@/components/CountCard.vue'

function onReceived(response, props) {
  props.text = response.total.toLocaleString() + ' Repositories'
}
</script>

<template>
  <CountCard url="/api/stats/count/repositories" @received="onReceived" color="lime" icon="database" />
</template>

<style scoped></style>
