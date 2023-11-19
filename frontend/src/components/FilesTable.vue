<script setup>
import { ref, watch } from 'vue'
import { sortParams } from '@/components/utils'
import { filters } from '@/utils/filters'
import DataGrid from '@/components/DataGrid.vue'
import _ from 'lodash'
import moment from 'moment/moment'

const props = defineProps({
  size: {
    type: String,
    default: 'lg',
  },
})

const grid = ref(null)

const columns = [
  {
    name: 'Path',
    field: 'path',
    type: 'text',
    size: 'l',
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        onClick: (v) => {
          filters.data.file = v.path
        },
      },
    ],
  },
  {
    name: 'Project',
    field: 'project.name',
    type: 'text',
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        show: (v) => v.project,
        onClick: (v) => {
          filters.data.project = v.project.name
        },
      },
    ],
  },
  {
    name: 'Repo',
    field: 'repo.name',
    type: 'text',
    show: props.size === 'lg',
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        show: (v) => v.repo,
        onClick: (v) => {
          filters.data.repo = v.repo.name
        },
      },
    ],
  },
  {
    name: 'Lines',
    field: 'size.lines',
    type: 'int',
  },
  {
    name: 'Changes 6m',
    field: 'changes.in6Months',
    show: props.size === 'lg',
    type: 'int',
  },
  {
    name: 'Changes total',
    field: 'changes.total',
    show: props.size === 'lg',
    type: 'int',
  },
  {
    name: 'First seen',
    field: 'firstSeen',
    type: 'date',
    show: props.size === 'lg',
  },
  {
    name: 'Last seen',
    field: 'lastSeen',
    type: 'date',
    show: props.size === 'lg',
  },
  {
    name: 'Exists',
    field: 'exists',
    type: 'text',
    format: (v) => (v.exists ? 'Yes' : 'No'),
  },
]

async function loadPage(page, pageSize, sort, asc) {
  let s = sortParams(page, pageSize, sort, asc)
  let f = filters.toQueryString({ file: 'q', repo: 'repo', person: 'person' })

  return await window.api.get(`/api/files?${f}&${s}`)
}

async function loadChart() {
  const response = await Promise.all([
    window.api.get('/api/stats/seen/files?' + filters.toQueryString({ file: 'q', repo: 'repo', person: 'person' })),
  ])

  const labels = []
  const filesSum = []
  const filesAdd = []
  const filesDel = []
  let filesPrev = 0

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
  let now = moment().format(format)
  for (let i = min; i <= max; i = i.add(1, 'month')) {
    let month = i.format(format)

    labels.push(month + '-15')

    let fs = response[0][month] || {}

    filesPrev += fs.firstSeen || 0
    filesSum.push(filesPrev)
    filesPrev -= fs.lastSeen || 0

    filesAdd.push(fs.firstSeen || 0)
    filesDel.push(month === now ? 0 : -(fs.lastSeen || 0))
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
      stacked: true,
    },
    fill: {
      opacity: [1, 0.2, 0.2],
    },
    stroke: {
      width: [2, 0, 0],
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
        seriesName: 'Files added',
        labels: {
          padding: 4,
          formatter: function (value) {
            return value.toLocaleString()
          },
        },
      },
      {
        show: false,
        seriesName: 'Files added',
        labels: {
          padding: 4,
          formatter: function (value) {
            return Math.abs(value).toLocaleString()
          },
        },
      },
    ],
    labels: labels,
    colors: [tabler.getColor('blue'), tabler.getColor('green'), tabler.getColor('red')],
    legend: {
      show: false,
    },
  }

  const series = [
    {
      name: 'Files total',
      type: 'line',
      data: filesSum,
    },
    {
      name: 'Files added',
      type: 'column',
      data: filesAdd,
    },
    {
      name: 'Files deleted',
      type: 'column',
      data: filesDel,
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
    title="Files"
    :columns="columns"
    :pageSize="size === 'md' ? 5 : null"
    :loadPage="size !== 'sm' ? loadPage : null"
    :loadChart="loadChart"
  />
</template>

<style scoped></style>
