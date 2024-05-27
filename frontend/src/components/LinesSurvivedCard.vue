<script setup>
import _ from 'lodash'
import moment from 'moment'
import { onMounted, reactive, ref, watch } from 'vue'
import { filters } from '@/utils/filters'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'

const props = defineProps({
  personId: Number,
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

  card.value.request(`/api/stats/survived/lines?${f}`, function (response) {
    const labels = []
    const code = []
    const comment = []
    const blank = []
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
      if (!data.code) data.code = 0
      if (!data.comment) data.comment = 0
      if (!data.blank) data.blank = 0
      sum += data.code + data.comment + data.blank

      code.push(data.code)
      comment.push(data.comment)
      blank.push(data.blank)
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
          seriesName: 'Code',
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
            },
          },
        },
        {
          seriesName: 'Code',
          show: false,
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
            },
          },
        },
        {
          seriesName: 'Code',
          show: false,
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
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
      colors: [tabler.getColor('blue'), '#1ab7ea', tabler.getColor('gray-300')],
      legend: {
        show: false,
      },
    }

    data.series = [
      {
        name: 'Code',
        type: 'bar',
        data: code,
      },
      {
        name: 'Comments',
        type: 'bar',
        data: comment,
      },
      {
        name: 'Blank lines',
        type: 'bar',
        data: blank,
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
      <h3 class="card-title">Lines that survived</h3>
    </div>

    <div class="card-body">
      <div class="chart-lg">
        <apexchart type="bar" height="240" :options="data.opts" :series="data.series" />
      </div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
