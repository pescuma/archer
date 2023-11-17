<script setup>
import { ref, watch } from 'vue'
import DataGrid from '@/components/DataGrid.vue'
import { sortParams } from './utils'
import { filters } from '@/utils/filters'

const grid = ref(null)

const columns = [
  {
    name: 'Date',
    field: 'date',
    type: 'date',
  },
  {
    name: 'Repo',
    field: 'repo.name',
    type: 'text',
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        onClick: function (v) {
          filters.data.repo = v.repo.name
        },
      },
    ],
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
    actions: [
      {
        name: 'Merge commit',
        icon: 'git-merge',
        before: true,
        show: (v) => v.parents.length > 1,
      },
    ],
  },
  {
    name: 'Author',
    field: 'author.name',
    type: 'text',
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        onClick: function (v) {
          filters.data.person = v.author.name
        },
      },
    ],
  },
  {
    name: 'Committer',
    field: 'committer.name',
    type: 'text',
    format: (v) => {
      if (v.committer.id === v.author.id) return ''
      return v.committer.name
    },
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        show: (v) => v.committer.id !== v.author.id,
        onClick: function (v) {
          filters.data.person = v.author.name
        },
      },
    ],
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

const actions = [
  {
    name: 'Ignore',
    icon: 'circle-minus',
    onClick: async function (commit) {
      await window.api.patch(`/api/repos/${commit.repo.id}/commits/${commit.id}`, { ignore: true })
    },
  },
]

async function loadPage(page, pageSize, sort, asc) {
  let s = sortParams(page, pageSize, sort, asc)
  let f = filters.toQueryString({ repo: 'repo', person: 'person' })

  return await window.api.get(`/api/commits?${f}&${s}`)
}

watch(
  () => filters.data,
  () => grid.value.refresh(),
  { deep: true }
)
</script>

<template>
  <DataGrid ref="grid" title="Commits" :columns="columns" :actions="actions" :loadPage="loadPage" />
</template>

<style scoped></style>
