<script setup>
import _ from 'lodash'
import moment from 'moment'
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'
import PaginationCardFooter from '@/components/PaginationCardFooter.vue'

const card = ref(null)

const props = defineProps({
  title: String,
  columns: Array,
  actions: Array,
  pageSize: Number,
  loadPage: Function,
  loadChart: Function,
})

const data = reactive({
  count: 0,
  sort: '',
  asc: true,
  pageNum: -1,
  pageRows: [],
})

const page = computed(() => {
  return {
    num: data.pageNum,
    size: props.pageSize || 10,
    rows: data.pageRows,
  }
})

const columns = computed(() => {
  let result = []

  for (let c of props.columns) {
    let rc = {
      name: c.name,
      show: c.show === undefined || c.show === null ? true : c.show,
      sort: c.field,
      size: c.size,
      actions: prepareActions(c.actions),
      th_class: '',
      td_class: '',
      style: '',
      defaultAsc: true,
      right: false,
      format: function (r) {
        if (c.format) return c.format(r)

        let v = _.get(r, c.field)
        if (v === null || v === undefined) return ''

        return v
      },
      tooltip: function (r) {
        if (c.tooltip) return c.tooltip(r)

        return undefined
      },
    }

    switch (c.type) {
      case 'text': {
        let fullFormat = rc.format
        rc.format = function (r) {
          let v = '' + fullFormat(r)
          return _.chain(v.split('\n'))
            .map((s) => _.trim(s))
            .filter((s) => s.length > 0)
            .first()
            .value()
        }
        rc.tooltip = function (r) {
          if (c.tooltip) return c.tooltip(r)

          let v = '' + fullFormat(r)
          if (rc.format(r) !== v) return v
          else return undefined
        }
        break
      }
      case 'int': {
        rc.th_class = 'w-1 text-end'
        rc.right = true
        rc.defaultAsc = false
        rc.format = function (r) {
          if (c.format) return c.format(r)

          let v = _.get(r, c.field)
          if (v === null || v === undefined) return ''

          return Math.round(v).toLocaleString()
        }
        break
      }
      case 'float': {
        rc.th_class = 'w-1 text-end'
        rc.defaultAsc = false
        rc.right = true
        rc.format = function (r) {
          if (c.format) return c.format(r)

          let v = _.get(r, c.field)
          if (v === null || v === undefined) return ''

          return Math.round(Math.round(v * 100) / 100).toLocaleString()
        }
        break
      }
      case 'date': {
        rc.th_class = 'w-1 text-end'
        rc.defaultAsc = false
        rc.right = true
        rc.format = function (r) {
          if (c.format) return c.format(r)

          let v = _.get(r, c.field)
          if (v === null || v === undefined) return ''

          return moment(v).toDate().toLocaleDateString()
        }
        break
      }
      case 'datetime': {
        rc.size = 'l'
        rc.th_class = 'w-1'
        rc.defaultAsc = false
        rc.right = true
        rc.format = function (r) {
          if (c.format) return c.format(r)

          let v = _.get(r, c.field)
          if (v === null || v === undefined) return ''

          return moment(v).toDate().toLocaleString()
        }
        break
      }
      default: {
        throw 'Invalid column type: ' + c.type
      }
    }

    switch (rc.size) {
      case 's':
        rc.th_class += ' w-1'
        rc.style = 'max-width: 60px'
        break
      case 'l':
        rc.style = 'max-width: 240px'
        break
      case 'xl':
        rc.style = 'max-width: 360px'
        break
      default:
        rc.style = 'max-width: 120px'
        break
    }

    result.push(rc)
  }

  return result
})

const actions = computed(() => {
  return prepareActions(props.actions)
})

function prepareActions(actions) {
  const result = []

  for (const a of actions || []) {
    if (typeof a.show === 'boolean' && !a.show) continue

    let ra = {
      name: a.name,
      icon: 'icon-' + a.icon,
      class: 'text-decoration-none',
      show: typeof a.show === 'function' ? a.show : () => true,
      onClick: a.onClick || (() => {}),
    }

    if (!a.onClick) {
      ra.class += ' cursor-default'
    }

    result.push(ra)
  }

  return result
}

async function loadPage(pageNum, sort, asc) {
  let result = await card.value.loading(async function () {
    return await props.loadPage(pageNum, page.value.size, sort, asc)
  })

  data.count = result.total
  data.pageRows = result.data
  data.pageNum = pageNum
  data.sort = sort
  data.asc = asc

  await nextTick()

  document.querySelectorAll('.text-truncate').forEach(function (e) {
    if (!e.title && e.textContent && e.offsetWidth < e.scrollWidth) {
      e.title = e.textContent
    }
  })
}

function onPage(p) {
  loadPage(p, data.sort, data.asc)
}

function onSort(field) {
  if (field === data.sort) {
    loadPage(page.value.num, data.sort, !data.asc)
  } else {
    let c = _.chain(columns.value)
      .filter((c) => c.sort === field)
      .first()
      .value()
    loadPage(page.value.num, c.sort, c.defaultAsc)
  }
}

const chart = reactive({
  opts: {},
  series: [],
})

async function loadChart() {
  if (!props.loadChart) return

  let result = await card.value.loading(async function () {
    return await props.loadChart()
  })

  chart.opts = result.opts
  chart.series = result.series
}

function refresh() {
  if (data.pageNum === -1) {
    let c = columns.value[0]
    data.pageNum = 1
    data.sort = c.sort
    data.asc = c.defaultAsc
  }

  loadChart()
  loadPage(data.pageNum, data.sort, data.asc)
}

onMounted(async function () {
  refresh()
})

defineExpose({ refresh })
</script>

<template>
  <CardWithPlaceholder ref="card" type="table">
    <div class="card-header">
      <h3 class="card-title">{{ props.title }}</h3>
    </div>

    <div class="card-body" v-if="chart.series.length > 0">
      <div class="chart-lg">
        <apexchart type="line" height="240" :options="chart.opts" :series="chart.series" />
      </div>
    </div>

    <div class="table-responsive border-bottom-0">
      <table class="card-table table table-vcenter text-nowrap">
        <thead>
          <tr>
            <template v-for="c in columns">
              <th v-if="c.show" @click.prevent="onSort(c.sort)" :class="c.th_class">
                {{ c.name }}
                <icon-chevron-up class="icon icon-sm icon-thick" v-if="c.sort === data.sort && data.asc" />
                <icon-chevron-down class="icon icon-sm icon-thick" v-if="c.sort === data.sort && !data.asc" />
              </th>
            </template>
            <th v-if="actions.length > 0" class="w-1"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in page.rows">
            <template v-for="c in columns">
              <td v-if="c.show" :class="c.td_class" :style="c.style">
                <div class="row">
                  <div :class="'col-auto text-truncate text-nowrap ' + (c.right ? ' ms-auto' : ' me-auto')" :title="c.tooltip(r)">
                    <template v-for="a in c.actions">
                      <a
                        v-if="a.before && a.show(r)"
                        href="#"
                        :class="a.class"
                        style="position: relative; z-index: 1"
                        :title="a.name"
                        @click.prevent="a.onClick(r)"
                      >
                        <component :is="a.icon" class="icon icon-sm align-text-bottom" />
                      </a>
                    </template>
                    <template v-for="a in c.actions">
                      <a
                        v-if="!a.before && a.show(r)"
                        href="#"
                        :class="a.class + ' float-end'"
                        style="position: relative; z-index: 1"
                        :title="a.name"
                        @click.prevent="a.onClick(r)"
                      >
                        <component :is="a.icon" class="icon icon-sm align-text-bottom" />
                      </a>
                    </template>

                    {{ c.format(r) }}
                  </div>
                </div>
              </td>
            </template>

            <td v-if="actions.length > 0">
              <template v-for="a in actions">
                <a v-if="a.show(r)" href="#" :class="a.class" :title="a.name" @click.prevent="a.onClick(r)">
                  <component :is="a.icon" class="icon" />
                </a>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationCardFooter :count="data.count" :page="page.num" :page-size="page.size" @pageChange="onPage" />
  </CardWithPlaceholder>
</template>

<style scoped></style>
