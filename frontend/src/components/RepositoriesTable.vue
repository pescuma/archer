<script setup>
import axios from 'axios'
import DataGrid from '@/components/DataGrid.vue'

const columns = [
  {
    name: 'Name',
    field: 'name',
    type: 'text',
  },
  {
    name: 'VCS',
    field: 'vcs',
    type: 'text',
  },
  {
    name: 'Commits',
    field: 'commitsTotal',
    type: 'int',
  },
  {
    name: 'Files',
    field: 'filesTotal',
    type: 'int',
  },
  {
    name: 'First commit',
    field: 'firstSeen',
    type: 'date',
  },
]

async function loadRowCount() {
  let result = await axios.get('/api/stats/count/repos')
  return result.data.total
}

async function loadPage(page, pageSize, sort, asc) {
  let result = await axios.get(`/api/repos?sort=${sort}&asc=${asc}&offset=${(page - 1) * pageSize}&limit=${pageSize}`)
  return result.data
}
</script>

<template>
  <DataGrid title="Repositories" :columns="columns" :loadRowCount="loadRowCount" :loadPage="loadPage" />
</template>

<style scoped></style>
