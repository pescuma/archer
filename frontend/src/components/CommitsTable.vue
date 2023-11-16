<script setup>
import DataGrid from '@/components/DataGrid.vue'

const columns = [
  {
    name: 'Date',
    field: 'date',
    type: 'datetime',
  },
  {
    name: 'Repository',
    field: 'repo.name',
    type: 'text',
    size: 's',
  },
  {
    name: 'Hash',
    field: 'hash',
    type: 'text',
    format: (v) => v.hash.slice(0, 7),
    tooltip: (v) => v.hash,
  },
  {
    name: 'Message',
    field: 'message',
    type: 'text',
    size: 'l',
  },
  {
    name: 'Committer',
    field: 'committer.name',
    type: 'text',
  },
  {
    name: 'Modified',
    field: 'modifiedLines',
    type: 'int',
  },
  {
    name: 'Added',
    field: 'addedLines',
    type: 'int',
  },
  {
    name: 'Deleted',
    field: 'deletedLines',
    type: 'int',
  },
  {
    name: 'Survived',
    field: 'survivedLines',
    type: 'int',
  },
]

async function loadPage(page, pageSize, sort, asc) {
  return await window.api.get(`/api/commits?sort=${sort}&asc=${asc}&offset=${(page - 1) * pageSize}&limit=${pageSize}`)
}
</script>

<template>
  <DataGrid title="Commits" :columns="columns" :loadPage="loadPage" />
</template>

<style scoped></style>
