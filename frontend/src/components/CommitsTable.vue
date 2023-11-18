<script setup>
import _ from 'lodash'
import moment from 'moment'
import { ref, watch } from 'vue'
import { sortParams } from './utils'
import { filters } from '@/utils/filters'
import DataGrid from '@/components/DataGrid.vue'

const props = defineProps({
  type: String,
})

const grid = ref(null)

const columns = [
  {
    name: 'Date',
    field: 'date',
    type: 'date',
    tooltip: (v) => moment(v.date).toDate().toLocaleString(),
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
    show: props.type !== 'sm',
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
        show: (v) => v.parents && v.parents.length > 1,
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
    show: props.type !== 'sm',
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
    show: props.type !== 'sm',
  },
  {
    name: 'Added',
    field: 'addedLines',
    type: 'int',
    show: props.type !== 'sm',
  },
  {
    name: 'Deleted',
    field: 'deletedLines',
    type: 'int',
    show: props.type !== 'sm',
  },
  {
    name: 'Survived',
    field: 'survivedLines',
    type: 'int',
    show: props.type !== 'sm',
  },
]

const actions = [
  {
    name: 'Ignore',
    icon: 'circle-minus',
    show: props.type !== 'sm',
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

async function loadChart() {
  const response = await Promise.all([
    window.api.get('/api/stats/seen/repos?' + filters.toQueryString({ repo: 'q', person: 'person' })),
    window.api.get('/api/stats/seen/commits?' + filters.toQueryString({ repo: 'repo', person: 'person' })),
  ])

  const labels = []
  const repos = []
  const commitsSum = []
  const commitsMonth = []
  let reposPrev = 0
  let commitsPrev = 0

  let format = 'YYYY-MM'
  let months = _.chain(response)
    .flatMap(function (i) {
      return _.keys(i)
    })
    .filter(function (i) {
      return i !== '0001-01'
    })
    .concat(moment().format(format))
  let min = moment(months.min(), format)
  let max = moment(months.max(), format)
  for (let i = min; i <= max; i = i.add(1, 'month')) {
    let month = i.format(format)

    labels.push(month + '-15')

    reposPrev += response[0][month] || 0
    repos.push(reposPrev)

    let cm = response[1][month] || 0

    commitsPrev += cm
    commitsSum.push(commitsPrev)

    commitsMonth.push(cm)
  }

  const opts = {
    chart: {
      fontFamily: 'inherit',
      parentHeightOffset: 0,
      toolbar: {
        show: false,
      },
      zoom: {
        enabled: false,
      },
      animations: {
        enabled: false,
      },
    },
    fill: {
      opacity: [1, 0.2, 1],
    },
    stroke: {
      width: [2, 0, 0.5],
      lineCap: 'round',
      curve: 'smooth',
    },
    tooltip: {
      theme: 'dark',
      shared: true,
      intersect: false,
    },
    grid: {
      padding: {
        top: -20,
        right: 0,
        left: -4,
        bottom: -4,
      },
      strokeDashArray: 4,
    },
    xaxis: {
      labels: {
        padding: 0,
        formatter: function (value) {
          return moment(value).format('YYYY-MM')
        },
      },
      tooltip: {
        enabled: false,
      },
      axisBorder: {
        show: false,
      },
      type: 'datetime',
    },
    yaxis: [
      {
        labels: {
          padding: 4,
          formatter: function (value) {
            return value.toLocaleString()
          },
        },
      },
      {
        show: false,
        labels: {
          padding: 4,
          formatter: function (value) {
            return value.toLocaleString()
          },
        },
      },
      {
        opposite: true,
        show: false,
        labels: {
          padding: 4,
          formatter: function (value) {
            return value.toLocaleString()
          },
        },
      },
    ],
    labels: labels,
    colors: [tabler.getColor('azure'), tabler.getColor('azure'), tabler.getColor('lime')],
    legend: {
      show: false,
    },
  }

  const series = [
    {
      name: 'Commits total',
      type: 'line',
      data: commitsSum,
    },
    {
      name: 'Commits',
      type: 'column',
      data: commitsMonth,
    },
    {
      name: 'Repositories total',
      type: 'line',
      data: repos,
    },
  ]

  return {
    opts,
    series,
  }
}

watch(
  () => filters.data,
  () => grid.value.refresh(),
  { deep: true }
)
</script>

<template>
  <DataGrid
    ref="grid"
    title="Commits"
    :columns="columns"
    :actions="actions"
    :pageSize="type === 'sm' ? 5 : null"
    :loadPage="loadPage"
    :loadChart="loadChart"
  />
</template>

<style scoped></style>
