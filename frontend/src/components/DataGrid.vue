<script setup>
import _ from 'lodash'
import moment from 'moment/moment'
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'
import PaginationCardFooter from '@/components/PaginationCardFooter.vue'

const card = ref(null)

const props = defineProps({
  title: String,
  columns: Array,
  actions: Array,
  loadPage: Function,
})

const data = reactive({
  count: 0,
  sort: '',
  asc: true,
  page: -1,
  pageSize: 10,
  pageRows: [],
})

const columns = computed(() => {
  let result = []

  for (let c of props.columns) {
    let rc = {
      name: c.name,
      sort: c.field,
      size: c.size,
      actions: c.actions,
      th_class: '',
      td_class: '',
      style: '',
      defaultAsc: true,
      format: function (r) {
        if (c.format) return c.format(r)
        else return _.get(r, c.field)
      },
      tooltip: function (r) {
        if (c.tooltip) return c.tooltip(r)
        else return undefined
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
        rc.td_class = 'text-end'
        rc.defaultAsc = false
        rc.format = function (r) {
          if (c.format) return c.format(r)
          else return Math.round(_.get(r, c.field)).toLocaleString()
        }
        break
      }
      case 'float': {
        rc.th_class = 'w-1 text-end'
        rc.td_class = 'text-end'
        rc.defaultAsc = false
        rc.format = function (r) {
          if (c.format) return c.format(r)
          return Math.round(Math.round(_.get(r, c.field) * 100) / 100).toLocaleString()
        }
        break
      }
      case 'date': {
        rc.th_class = 'w-1 text-end'
        rc.td_class = 'text-end'
        rc.defaultAsc = false
        rc.format = function (r) {
          if (c.format) return c.format(r)
          else return moment(_.get(r, c.field)).toDate().toLocaleDateString()
        }
        break
      }
      case 'datetime': {
        rc.size = 'l'
        rc.th_class = 'w-1'
        rc.defaultAsc = false
        rc.format = function (r) {
          if (c.format) return c.format(r)
          else return moment(_.get(r, c.field)).toDate().toLocaleString()
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

async function loadPage(page, sort, asc) {
  let result = await card.value.loading(async function () {
    return await props.loadPage(page, data.pageSize, sort, asc)
  })

  data.count = result.total
  data.pageRows = result.data
  data.page = page
  data.sort = sort
  data.asc = asc

  await nextTick()

  document.querySelectorAll('.text-truncate').forEach(function (e) {
    if (!e.title && e.textContent && e.offsetWidth < e.scrollWidth) {
      e.title = e.textContent
    }
  })
}

function page(p) {
  loadPage(p, data.sort, data.asc)
}

function sort(field) {
  if (field === data.sort) {
    loadPage(data.page, data.sort, !data.asc)
  } else {
    let c = _.chain(columns.value)
      .filter((c) => c.sort === field)
      .first()
      .value()
    loadPage(data.page, c.sort, c.defaultAsc)
  }
}

function click(r, a) {
  a.onClick(r)
}

function refresh() {
  loadPage(data.page, data.sort, data.asc)
}

onMounted(async function () {
  let c = columns.value[0]
  await loadPage(1, c.sort, c.defaultAsc)
})

defineExpose({ refresh })
</script>

<template>
  <CardWithPlaceholder ref="card" type="table">
    <div class="card-header">
      <h3 class="card-title">{{ props.title }}</h3>
    </div>

    <div class="table-responsive border-bottom-0">
      <table class="card-table table table-vcenter text-nowrap">
        <thead>
          <tr>
            <th v-for="c in columns" :class="c.th_class" @click.prevent="sort(c.sort)">
              {{ c.name }}
              <icon-chevron-up class="icon icon-sm icon-thick" v-if="c.sort === data.sort && data.asc" />
              <icon-chevron-down class="icon icon-sm icon-thick" v-if="c.sort === data.sort && !data.asc" />
            </th>
            <th v-if="actions" class="w-1"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in data.pageRows">
            <td v-for="c in columns" :class="c.td_class" :style="c.style">
              <div class="row">
                <div class="col-auto text-truncate text-nowrap" :title="c.tooltip(r)">
                  <a
                    v-for="a in c.actions"
                    href="#"
                    class="text-decoration-none float-end ms-1"
                    :title="a.name"
                    @click.prevent="click(r, a)"
                  >
                    <component :is="'icon-' + a.icon" class="icon icon-sm align-text-bottom" />
                  </a>

                  {{ c.format(r) }}
                </div>
              </div>
            </td>
            <td v-if="actions">
              <a v-for="a in actions" href="#" class="text-decoration-none" :title="a.name" @click.prevent="click(r, a)">
                <component :is="'icon-' + a.icon" class="icon" />
              </a>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationCardFooter :count="data.count" :page="data.page" :page-size="data.pageSize" @pageChange="page" />
  </CardWithPlaceholder>
</template>

<style scoped></style>
