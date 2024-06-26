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
    name: 'Names',
    field: 'names',
    type: 'text',
    format: (v) => _.join(v.names, ', '),
    tooltip: (v) => _.join(v.names, '\n'),
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
    show: props.size === 'lg',
    format: (v) => _.join(v.emails, ', '),
    tooltip: (v) => _.join(v.emails, '\n'),
  },
  {
    name: 'Commits (total)',
    field: 'changes.total',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Commits (6 months)',
    field: 'changes.in6Months',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Modified lines',
    field: 'changes.linesModified',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Added lines',
    field: 'changes.linesAdded',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Deleted lines',
    field: 'changes.linesDeleted',
    type: 'int',
    show: props.size === 'lg',
  },
  {
    name: 'Survived lines',
    field: 'blame.total',
    type: 'int',
    show: props.size === 'lg',
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
]

async function loadPage(page, pageSize, sort, asc) {
  let s = sortParams(page, pageSize, sort, asc)
  let f = filters.toQueryString()

  return await window.api.get(`/api/people?${f}&${s}`)
}

async function loadChart() {
  const response = await window.api.get('/api/stats/seen/people?' + filters.toQueryString())

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
  let limitLastSeen = moment().subtract(2, 'months').format(format)
  for (let i = min; i <= max; i = i.add(1, 'month')) {
    let month = i.format(format)

    labels.push(month + '-15')

    let fs = response[month] || {}

    prev += fs.firstSeen || 0
    sum.push(prev)
    if (month <= limitLastSeen) {
      prev -= fs.lastSeen || 0
    }

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
        seriesName: 'People added',
        labels: {
          padding: 4,
          formatter: function (value) {
            return value.toLocaleString()
          },
        },
      },
      {
        show: false,
        seriesName: 'People added',
        labels: {
          padding: 4,
          formatter: function (value) {
            return Math.abs(value).toLocaleString()
          },
        },
      },
    ],
    labels: labels,
    colors: [tabler.getColor('purple'), tabler.getColor('green'), tabler.getColor('red')],
    legend: {
      show: false,
    },
  }

  const series = [
    {
      name: 'People total',
      type: 'line',
      data: sum,
    },
    {
      name: 'People added',
      type: 'column',
      data: add,
    },
    {
      name: 'People deleted',
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
    title="People"
    :columns="columns"
    :pageSize="size === 'md' ? 5 : null"
    :loadPage="size !== 'sm' ? loadPage : null"
    :loadChart="loadChart"
  />
</template>

<style scoped></style>
