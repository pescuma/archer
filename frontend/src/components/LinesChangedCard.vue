<script setup>
import _ from 'lodash'
import moment from 'moment/moment'
import { onMounted, reactive, ref, watch } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'
import { filters } from '@/utils/filters'

const card = ref(null)

const data = reactive({
  opts: {},
  series: [],
})

onMounted(refresh)

watch(() => filters.data, refresh, { deep: true })

function refresh() {
  let f = filters.toQueryString({ repo: 'repo.name', person: 'person' })

  card.value.request(`/api/stats/changed/lines?${f}`, function (response) {
    const labels = []
    const modified = []
    const added = []
    const deleted = []

    let format = 'YYYY-MM'
    let months = _.chain(response)
      .keys()
      .filter(function (i) {
        return i !== '0001-01'
      })
      .concat(moment().format(format))
    let min = moment(months.min(), format)
    let max = moment(months.max(), format)
    for (let i = min; i <= max; i = i.add(1, 'month')) {
      let month = i.format(format)

      labels.push(month + '-15')

      let data = response[month] || { modified: 0, added: 0, deleted: 0 }
      modified.push(data.modified)
      added.push(data.added)
      deleted.push(-data.deleted)
    }

    data.opts = {
      chart: {
        fontFamily: 'inherit',
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
      plotOptions: {
        bar: {
          columnWidth: '50%',
        },
      },
      dataLabels: {
        enabled: false,
      },
      fill: {
        opacity: 1,
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
        crosshairs: {
          enabled: false,
          width: 'barWidth',
        },
        type: 'datetime',
      },
      yaxis: {
        labels: {
          padding: 4,
          formatter: function (value) {
            return Math.abs(value).toLocaleString()
          },
        },
      },
      labels: labels,
      colors: [tabler.getColor('blue'), tabler.getColor('green'), tabler.getColor('red')],
      legend: {
        show: false,
      },
    }

    data.series = [
      {
        name: 'Modified',
        data: modified,
      },
      {
        name: 'Added',
        data: added,
      },
      {
        name: 'Deleted',
        data: deleted,
      },
    ]
  })
}
</script>

<template>
  <CardWithPlaceholder ref="card" type="chart">
    <div class="card-header">
      <h3 class="card-title">Lines changed</h3>
    </div>

    <div class="card-body">
      <div class="chart-lg">
        <apexchart type="bar" height="240" :options="data.opts" :series="data.series" />
      </div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
