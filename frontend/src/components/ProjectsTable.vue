<script setup>
import _ from 'lodash'
import moment from 'moment/moment'
import { ref, watch } from 'vue'
import { sortParams } from '@/components/utils'
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
    name: 'Name',
    field: 'name',
    type: 'text',
    size: 'l',
    actions: [
      {
        name: 'Filter',
        icon: 'filter',
        onClick: (v) => {
          filters.data.proj = v.name
        },
      },
    ],
  },
  {
    name: 'Type',
    field: 'type',
    type: 'text',
  },
  {
    name: 'Dir',
    field: 'rootDir',
    type: 'text',
    show: props.size === 'lg',
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
    name: 'Files',
    field: 'size.files',
    type: 'int',
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
]

async function loadPage(page, pageSize, sort, asc) {
  let s = sortParams(page, pageSize, sort, asc)
  let f = filters.toQueryString()

  return await window.api.get(`/api/projects?${f}&${s}`)
}

async function loadChart() {
  const response = await window.api.get('/api/stats/seen/projects?' + filters.toQueryString())

  const labels = []
  const sum = []
  const add = []
  const del = []
  let prev = 0

  let format = 'YYYY-MM'
  let months = _.chain(_.keys(response))
    .filter(function (i) {
      return i !== '0001-01'
    })
    .concat(moment().format(format))
  let min = moment(months.min().value(), format)
  let max = moment(months.max().value(), format)
  let now = moment().format(format)
  for (let i = min; i <= max; i = i.add(1, 'month')) {
    let month = i.format(format)

    labels.push(month + '-15')

    let fs = response[month] || {}

    prev += fs.firstSeen || 0
    sum.push(prev)
    prev -= fs.lastSeen || 0

    add.push(fs.firstSeen || 0)
    del.push(month === now ? 0 : -(fs.lastSeen || 0))
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
        seriesName: 'Projects added',
        labels: {
          padding: 4,
          formatter: function (value) {
            return value.toLocaleString()
          },
        },
      },
      {
        show: false,
        seriesName: 'Projects added',
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
      name: 'Projects total',
      type: 'line',
      data: sum,
    },
    {
      name: 'Projects added',
      type: 'column',
      data: add,
    },
    {
      name: 'Projects deleted',
      type: 'column',
      data: del,
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
    title="Projects"
    :columns="columns"
    :pageSize="size === 'md' ? 5 : null"
    :loadPage="size !== 'sm' ? loadPage : null"
    :loadChart="loadChart"
  />
</template>

<style scoped></style>
