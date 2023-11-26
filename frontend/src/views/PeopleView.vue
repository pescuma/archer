<script setup>
import PeopleTable from '@/components/PeopleTable.vue'
import { reactive } from 'vue'
import PersonLinesCard from '@/components/PersonLinesCard.vue'

const data = reactive({
  rows: [],
})

function onRowsChanged(rows) {
  data.rows = rows
}
</script>

<template>
  <div class="col-12">
    <PeopleTable @rows-changed="onRowsChanged" />
  </div>

  <div class="col-12" v-for="row in data.rows">
    <PersonLinesCard :person="row" />
  </div>
</template>

<style scoped></style>
