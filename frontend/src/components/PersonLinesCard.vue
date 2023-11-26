<script setup>
import LinesChangedCard from '@/components/LinesChangedCard.vue'
import LinesSurvivedCard from '@/components/LinesSurvivedCard.vue'

defineProps({
  person: Object,
})
</script>

<template>
  <div class="card card-sm">
    <div class="card-header">
      <h3 class="card-title">{{ person.name }}</h3>
    </div>

    <div class="card-body">
      <div class="row">
        <div class="col-lg-6">
          <LinesChangedCard :person-id="person.id" />
        </div>
        <div class="col-lg-6">
          <LinesSurvivedCard :person-id="person.id" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
