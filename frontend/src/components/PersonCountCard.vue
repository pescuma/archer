<script setup>
import CountCard from '@/components/CountCard.vue'

function onReceived(response, props) {
  props.text = response.total.toLocaleString() + ' People'
}
</script>

<template>
  <CountCard url="/api/stats/count/people" @received="onReceived" color="purple" icon="users" />
</template>

<style scoped></style>
