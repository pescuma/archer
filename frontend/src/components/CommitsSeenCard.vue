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
  card.value.request(['/api/stats/seen/repos', '/api/stats/seen/commits'], function (response) {
    const labels = []
    const repos = []
    const commits = []
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

      commitsPrev += response[1][month] || 0
      commits.push(commitsPrev)
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
        width: [0.5, 2],
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
          opposite: true,
          show: false,
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
            },
          },
        },
        {
          labels: {
            padding: 4,
            formatter: function (value) {
              return value.toLocaleString()
            },
          },
        },
      ],
      labels: labels,
      colors: [tabler.getColor('lime'), tabler.getColor('cyan')],
      legend: {
        show: false,
      },
    }

    data.series = [
      {
        name: 'Repositories',
        data: repos,
      },
      {
        name: 'Commits',
        data: commits,
      },
    ]
  })
})
</script>

<template>
  <CardWithPlaceholder ref="card" type="chart">
    <h3 class="card-title">Commits</h3>

    <div class="chart-lg card-title">
      <apexchart type="line" height="240" :options="data.opts" :series="data.series" />
    </div>

    <div class="card-table border-top table-responsive">
      <table class="table table-vcenter">
        <thead>
          <tr>
            <th>User</th>
            <th>Commit</th>
            <th>Date</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td class="w-1">
              <span class="avatar avatar-sm" style="background-image: url(./static/avatars/000m.jpg)"></span>
            </td>
            <td class="td-truncate">
              <div class="text-truncate">Fix dart Sass compatibility (#29755)</div>
            </td>
            <td class="text-nowrap text-muted">28 Nov 2019</td>
          </tr>
          <tr>
            <td class="w-1">
              <span class="avatar avatar-sm">JL</span>
            </td>
            <td class="td-truncate">
              <div class="text-truncate">Change deprecated html tags to text decoration classes (#29604)</div>
            </td>
            <td class="text-nowrap text-muted">27 Nov 2019</td>
          </tr>
          <tr>
            <td class="w-1">
              <span class="avatar avatar-sm" style="background-image: url(./static/avatars/002m.jpg)"></span>
            </td>
            <td class="td-truncate">
              <div class="text-truncate">justify-content:between ⇒ justify-content:space-between (#29734)</div>
            </td>
            <td class="text-nowrap text-muted">26 Nov 2019</td>
          </tr>
          <tr>
            <td class="w-1">
              <span class="avatar avatar-sm" style="background-image: url(./static/avatars/003m.jpg)"></span>
            </td>
            <td class="td-truncate">
              <div class="text-truncate">Update change-version.js (#29736)</div>
            </td>
            <td class="text-nowrap text-muted">26 Nov 2019</td>
          </tr>
          <tr>
            <td class="w-1">
              <span class="avatar avatar-sm" style="background-image: url(./static/avatars/000f.jpg)"></span>
            </td>
            <td class="td-truncate">
              <div class="text-truncate">Regenerate package-lock.json (#29730)</div>
            </td>
            <td class="text-nowrap text-muted">25 Nov 2019</td>
          </tr>
        </tbody>
      </table>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
