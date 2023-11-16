<script setup>
import RepositoriesTable from '@/components/RepositoriesTable.vue'
import LinesSurvivedCard from '@/components/LinesSurvivedCard.vue'
import LinesChangedCard from '@/components/LinesChangedCard.vue'
import CommitsTable from '@/components/CommitsTable.vue'
</script>

<template>
  <div class="col-12">
    <LinesChangedCard />
  </div>
  <div class="col-12">
    <LinesSurvivedCard />
  </div>

  <div class="col-12">
    <RepositoriesTable />
  </div>

  <div class="col-12">
    <CommitsTable />
  </div>
</template>

<style scoped></style>
