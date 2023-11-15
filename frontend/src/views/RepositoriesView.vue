<script setup>
import RepositoriesTable from '@/components/RepositoriesTable.vue'
import LinesSurvivedCard from '@/components/LinesSurvivedCard.vue'
import LinesChangedCard from '@/components/LinesChangedCard.vue'
import { onMounted } from 'vue'

onMounted(function () {
  console.log('on mounted repositories view')
})
</script>

<template>
  <div class="page-body">
    <div class="container-xl">
      <div class="row row-deck row-cards">
        <div class="col-12">
          <LinesChangedCard />
        </div>
        <div class="col-12">
          <LinesSurvivedCard />
        </div>
        <div class="col-12">
          <RepositoriesTable />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
