<script setup>
import _ from 'lodash'
import moment from 'moment/moment'
import { computed, onMounted, reactive, ref } from 'vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'
import { IconChevronDown, IconChevronUp } from '@tabler/icons-vue'
import PaginationCardFooter from '@/components/PaginationCardFooter.vue'

const card = ref(null)

const props = defineProps({
  title: String,
  columns: Array,
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
    let r = {
      name: c.name,
      field: c.field,
      size: c.size,
      th_class: '',
      td_class: '',
      style: '',
      defaultAsc: true,
    }

    switch (c.type) {
      case 'text': {
        r.format = function (v) {
          return v
        }
        break
      }
      case 'int': {
        r.th_class = 'w-1 text-end'
        r.td_class = 'text-end'
        r.defaultAsc = false
        r.format = function (v) {
          return Math.round(v).toLocaleString()
        }
        break
      }
      case 'float': {
        r.th_class = 'w-1 text-end'
        r.td_class = 'text-end'
        r.defaultAsc = false
        r.format = function (v) {
          return Math.round(Math.round(v * 100) / 100).toLocaleString()
        }
        break
      }
      case 'date': {
        r.th_class = 'w-1 text-end'
        r.td_class = 'text-end'
        r.defaultAsc = false
        r.format = function (v) {
          return moment(v).toDate().toLocaleDateString()
        }
        break
      }
      case 'datetime': {
        r.th_class = 'w-1'
        r.defaultAsc = false
        r.format = function (v) {
          return moment(v).toDate().toLocaleString()
        }
        break
      }
      default: {
        throw 'Invalid column type: ' + c.type
      }
    }

    r.td_class += ' text-truncate'

    switch (c.size) {
      case 's':
        r.th_class += ' w-1'
        r.style = 'max-width: 100px'
        break
      case 'l':
        r.style = 'max-width: 400px'
        break
      default:
        r.style = 'max-width: 200px'
        break
    }

    result.push(r)
  }

  return result
})

async function loadPage(page, sort, asc) {
  if (page === data.page && sort === data.sort && asc === data.asc) return

  let result = await card.value.loading(async function () {
    return await props.loadPage(page, data.pageSize, sort, asc)
  })

  data.count = result.total
  data.pageRows = result.data
  data.page = page
  data.sort = sort
  data.asc = asc
}

function page(p) {
  loadPage(p, data.sort, data.asc)
}

function sort(field) {
  if (field === data.sort) {
    loadPage(data.page, data.sort, !data.asc)
  } else {
    let c = _.chain(columns.value)
      .filter((c) => c.field === field)
      .first()
      .value()
    loadPage(data.page, c.field, c.defaultAsc)
  }
}

onMounted(async function () {
  let c = columns.value[0]
  loadPage(1, c.field, c.defaultAsc)
})
</script>

<template>
  <CardWithPlaceholder ref="card" type="table">
    <div class="card-header">
      <h3 class="card-title">{{ props.title }}</h3>
    </div>

    <div class="table-responsive border-bottom-0">
      <table class="card-table table table-vcenter text-nowrap text-truncate">
        <thead>
          <tr>
            <th v-for="c in columns" :class="c.th_class" @click.prevent="sort(c.field)">
              {{ c.name }}
              <IconChevronUp class="icon icon-sm icon-thick" v-if="c.field === data.sort && data.asc" />
              <IconChevronDown class="icon icon-sm icon-thick" v-if="c.field === data.sort && !data.asc" />
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in data.pageRows">
            <td v-for="c in columns" :class="c.td_class" :style="c.style" :title="c.size === 'l' ? c.format(_.get(r, c.field)) : undefined">
              {{ c.format(_.get(r, c.field)) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationCardFooter :count="data.count" :page="data.page" :page-size="data.pageSize" @pageChange="page" />
  </CardWithPlaceholder>
</template>

<style scoped></style>
