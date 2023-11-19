<script setup>
import { ref, watch } from 'vue'
import { sortParams } from '@/components/utils'
import { filters } from '@/utils/filters'
import DataGrid from '@/components/DataGrid.vue'

const grid = ref(null)

const columns = [
  {
    name: 'Name',
    field: 'name',
    type: 'text',
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        onClick: function (v) {
          filters.data.repo = v.name
        },
      },
    ],
  },
  {
    name: 'VCS',
    field: 'vcs',
    type: 'text',
    actions: [
      {
        icon: 'brand-git',
        before: true,
      },
    ],
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
  let s = sortParams(page, pageSize, sort, asc)
  let f = filters.toQueryString({ repo: 'q', person: 'person' })

  return await window.api.get(`/api/repos?${f}&${s}`)
}

watch(
  () => filters.data,
  () => grid.value.refresh(),
  { deep: true }
)
</script>

<template>
  <DataGrid ref="grid" title="Repositories" :columns="columns" :loadPage="loadPage" />
</template>

<style scoped></style>
