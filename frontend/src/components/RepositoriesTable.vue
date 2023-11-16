<script setup>
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

async function loadPage(page, pageSize, sort, asc) {
  return await window.api.get(`/api/repos?sort=${sort}&asc=${asc}&offset=${(page - 1) * pageSize}&limit=${pageSize}`)
}
</script>

<template>
  <DataGrid title="Repositories" :columns="columns" :loadPage="loadPage" />
</template>

<style scoped></style>
