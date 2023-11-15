<script setup>
import _ from 'lodash'
import moment from 'moment/moment'
import { onMounted, reactive, ref } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'

const card = ref(null)

const data = reactive({
  opts: {},
  series: [],
})

onMounted(function () {
  card.value.request(['/api/stats/seen/files', '/api/stats/seen/projects'], function (response) {
    const labels = []
    const files = []
    const projs = []
    let filesPrev = 0
    let projsPrev = 0

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

      filesPrev += response[0][month] || 0
      files.push(filesPrev)

      projsPrev += response[1][month] || 0
      projs.push(projsPrev)
    }

    data.opts = {
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
        opacity: 1,
      },
      stroke: {
        width: [2, 0.5],
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
      colors: [tabler.getColor('blue'), tabler.getColor('azure')],
      legend: {
        show: false,
      },
    }

    data.series = [
      {
        name: 'Files',
        data: files,
      },
      {
        name: 'Projects',
        data: projs,
      },
    ]
  })
})
</script>

<template>
  <CardWithPlaceholder ref="card" type="chart">
    <div class="card-header">
      <h3 class="card-title">Files</h3>
    </div>

    <div class="card-body">
      <div class="chart-lg">
        <apexchart type="line" height="240" :options="data.opts" :series="data.series" />
      </div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
