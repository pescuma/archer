<script setup>
import _ from 'lodash'
import moment from 'moment'
import { onMounted, reactive, ref, watch } from 'vue'
import { filters } from '@/utils/filters'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'

const props = defineProps({
  personId: String,
})

const card = ref(null)

const data = reactive({
  opts: {},
  series: [],
})

onMounted(refresh)

if (props.personId) {
  watch(() => props.personId, refresh)
} else {
  watch(() => filters.data, refresh, { deep: true })
}

function refresh() {
  let f = filters.toQueryString()
  if (props.personId) {
    f += `&person.id=${encodeURIComponent(props.personId)}`
  }

  card.value.request(`/api/stats/changed/lines?${f}`, function (response) {
    const labels = []
    const modified = []
    const added = []
    const deleted = []
    const total = []

    let format = 'YYYY-MM'
    let months = _.chain(response)
      .keys()
      .filter(function (i) {
        return i !== '0001-01'
      })
      .concat(moment().format(format))
    let min = moment(months.min().value(), format)
    let max = moment(months.max().value(), format)
    let sum = 0
    for (let i = min; i <= max; i = i.add(1, 'month')) {
      let month = i.format(format)

      labels.push(month + '-15')

      let data = response[month] || {}
      if (!data.modified) data.modified = 0
      if (!data.added) data.added = 0
      if (!data.deleted) data.deleted = 0
      sum += data.added - data.deleted

      modified.push(data.modified)
      added.push(data.added)
      deleted.push(-data.deleted)
      total.push(sum)
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
      stroke: {
        width: 1,
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
        crosshairs: {
          enabled: false,
          width: 'barWidth',
        },
        type: 'datetime',
      },
      yaxis: [
        {
          seriesName: 'Modified',
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
            },
          },
        },
        {
          seriesName: 'Modified',
          show: false,
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
            },
          },
        },
        {
          seriesName: 'Modified',
          show: false,
          labels: {
            padding: 4,
            formatter: function (value) {
              return Math.abs(value).toLocaleString()
            },
          },
        },
        {
          seriesName: 'Total',
          show: false,
          opposite: true,
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
            },
          },
        },
      ],
      labels: labels,
      colors: [tabler.getColor('blue'), tabler.getColor('green'), tabler.getColor('red'), tabler.getColor('lime')],
      legend: {
        show: false,
      },
    }

    data.series = [
      {
        name: 'Modified',
        type: 'bar',
        data: modified,
      },
      {
        name: 'Added',
        type: 'bar',
        data: added,
      },
      {
        name: 'Deleted',
        type: 'bar',
        data: deleted,
      },
      {
        name: 'Total',
        type: 'line',
        data: total,
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
