<script setup>
import _ from 'lodash'
import moment from 'moment'
import { onMounted, reactive, ref } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'
import gravatarUrl from 'gravatar-url'

const card = ref(null)

const data = reactive({
  opts: {},
  series: [],
  commits: [],
})

onMounted(function () {
  card.value.request(['/api/stats/seen/repos', '/api/stats/seen/commits', '/api/commits?limit=5'], function (response) {
    data.commits = response[2].data

    const labels = []
    const repos = []
    const commits = []
    let reposPrev = 0
    let commitsPrev = 0

    let format = 'YYYY-MM'
    let months = _.chain(response)
      .take(2)
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

function createInitials(commit) {
  if (!commit.author) {
    return '?'
  }

  let initials = commit.author.name.split(' ').map(function (i) {
    return i[0]
  })

  if (initials.length > 2) {
    initials = [initials[0], initials[initials.length - 1]]
  }

  return initials.join('').toUpperCase()
}

function getGravatarStyle(commit) {
  if (!commit.author) {
    return ''
  }

  let candidates = commit.author.emails.find(function (i) {
    return !i.match(/@users.noreply.github.com$/)
  })
  if (!candidates) {
    return ''
  }

  return 'background-image: url(' + gravatarUrl(candidates[0], { default: 'blank' }) + ')'
}
</script>

<template>
  <CardWithPlaceholder ref="card" type="chart">
    <div class="card-header">
      <h3 class="card-title">Commits</h3>
    </div>

    <div class="card-body">
      <div class="chart-lg">
        <apexchart type="line" height="240" :options="data.opts" :series="data.series" />
      </div>
    </div>

    <div class="card-table border-top table-responsive">
      <table class="table table-vcenter">
        <thead>
          <tr>
            <th class="w-1">Author</th>
            <th class="w-2">Repo</th>
            <th>Message</th>
            <th>Date</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in data.commits" :key="c.id">
            <td>
              <span class="avatar avatar-sm" :style="getGravatarStyle(c)">{{ createInitials(c) }}</span>
            </td>
            <td>
              <div class="text-truncate">{{ c.repo.name }}</div>
            </td>
            <td class="td-truncate">
              <div class="text-truncate">{{ c.message }}</div>
            </td>
            <td class="text-truncate text-muted" :title="moment(c.date).toDate().toLocaleString()">
              {{ moment(c.date).toDate().toLocaleDateString() }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="card-footer"></div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
