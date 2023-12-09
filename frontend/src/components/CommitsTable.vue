<script setup>
import _ from 'lodash'
import moment from 'moment'
import { ref, watch } from 'vue'
import { sortParams } from './utils'
import { filters } from '@/utils/filters'
import DataGrid from '@/components/DataGrid.vue'

const props = defineProps({
  size: {
    type: String,
    default: 'lg',
  },
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
    show: props.size === 'lg',
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
    field: 'authors.name',
    type: 'text',
    format: (v) => _.chain(v.authors).map(a => a.name).join(", ").value(),
    tooltip: (v) => _.chain(v.authors).map(a => a.name).join("\n").value(),
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        onClick: function (v) {
          filters.data.person = v.authors[0].name
        },
      },
    ],
  },
  {
    name: 'Committer',
    field: 'committer.name',
    type: 'text',
    show: props.size === 'lg',
    format: (v) => {
      if (v.authors.length === 1 && v.committer.id === v.authors[0].id) return ''
      return v.committer.name
    },
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        show: (v) => !(v.authors.length === 1 && v.committer.id === v.authors[0].id),
        onClick: function (v) {
          filters.data.person = v.committer.name
        },
      },
    ],
  },
  {
    name: 'Modified',
    field: 'modifiedLines',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Added',
    field: 'addedLines',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Deleted',
    field: 'deletedLines',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Survived',
    field: 'blame',
    type: 'int',
    show: props.size === 'lg',
  },
]

const actions = [
  {
    name: 'Ignore',
    icon: 'circle-minus',
    show: props.size === 'lg',
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
    window.api.get('/api/stats/seen/repos?' + filters.toQueryString()),
    window.api.get('/api/stats/seen/commits?' + filters.toQueryString()),
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
  let min = moment(months.min().value(), format)
  let max = moment(months.max().value(), format)
  for (let i = min; i <= max; i = i.add(1, 'month')) {
    let month = i.format(format)

    labels.push(month + '-15')

    let rm = response[0][month] || 0
    reposPrev += rm.firstSeen || 0
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
      type: 'bar',
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
    :pageSize="size === 'md' ? 5 : null"
    :loadPage="size !== 'sm' ? loadPage : null"
    :loadChart="loadChart"
  />
</template>

<style scoped></style>
