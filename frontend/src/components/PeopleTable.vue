<script setup>
import _ from 'lodash'
import DataGrid from '@/components/DataGrid.vue'
import { ref, watch } from 'vue'
import { sortParams } from '@/components/utils'
import { filters } from '@/utils/filters'

const grid = ref(null)

const columns = [
  {
    name: 'Names',
    field: 'names',
    type: 'text',
    format: (v) => {
      let names = []
      names.push(v.name)
      for (let name of v.names) {
        if (name !== v.name) {
          names.push(name)
        }
      }
      return _.join(names, '\n')
    },
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        onClick: function (v) {
          filters.data.person = v.name
        },
      },
    ],
  },
  {
    name: 'Emails',
    field: 'emails',
    type: 'text',
    format: (v) => _.join(v.emails, ', '),
    tooltip: (v) => _.join(v.emails, '\n'),
  },
  {
    name: 'Commits (total)',
    field: 'changes.total',
    type: 'int',
  },
  {
    name: 'Commits (6 months)',
    field: 'changes.in6Months',
    type: 'int',
  },
  {
    name: 'Modified lines',
    field: 'changes.modifiedLines',
    type: 'int',
  },
  {
    name: 'Added lines',
    field: 'changes.addedLines',
    type: 'int',
  },
  {
    name: 'Deleted lines',
    field: 'changes.deletedLines',
    type: 'int',
  },
  {
    name: 'Survived lines',
    field: 'blame.lines',
    type: 'int',
  },
  {
    name: 'First seen',
    field: 'firstSeen',
    type: 'date',
  },
  {
    name: 'Last seen',
    field: 'lastSeen',
    type: 'date',
  },
]

async function loadPage(page, pageSize, sort, asc) {
  let s = sortParams(page, pageSize, sort, asc)
  let f = filters.toQueryString({ person: 'search' })

  return await window.api.get(`/api/people?${f}&${s}`)
}

watch(
  () => filters.data,
  () => grid.value.refresh(),
  { deep: true }
)
</script>

<template>
  <DataGrid ref="grid" title="People" :columns="columns" :loadPage="loadPage" />
</template>

<style scoped></style>
