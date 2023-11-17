<script setup>
import _ from 'lodash'
import moment from 'moment'
import { onMounted, reactive, ref } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'

const card = ref(null)

const data = reactive({
  opts: {},
  series: [],
})

onMounted(function () {
  card.value.request('/api/stats/seen/people', function (response) {
    const labels = []
    const people = []
    let peoplePrev = 0

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

      peoplePrev += response[month] || 0
      people.push(peoplePrev)
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
        width: 2,
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
      yaxis: {
        labels: {
          padding: 4,
          formatter: function (value) {
            return value.toLocaleString()
          },
        },
      },
      labels: labels,
      colors: [tabler.getColor('purple')],
      legend: {
        show: false,
      },
    }

    data.series = [
      {
        name: 'People',
        data: people,
      },
    ]
  })
})
</script>

<template>
  <CardWithPlaceholder ref="card" type="chart">
    <div class="card-header">
      <h3 class="card-title">People</h3>
    </div>

    <div class="card-body">
      <div class="chart-lg">
        <apexchart type="line" height="240" :options="data.opts" :series="data.series" />
      </div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
