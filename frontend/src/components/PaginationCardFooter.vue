<script setup>
import { computed } from 'vue'
import { IconChevronLeft, IconChevronRight } from '@tabler/icons-vue'

const emit = defineEmits(['pageChange'])

const data = defineProps({
  count: Number,
  page: Number,
  pageSize: Number,
})

const pagination = computed(() => {
  let result = {}
  result.start = Math.min((data.page - 1) * data.pageSize + 1, data.count)
  result.end = Math.min(data.page * data.pageSize, data.count)
  result.pageCount = Math.ceil(data.count / data.pageSize)
  result.hasPrev = data.page > 1
  result.hasNext = data.page < result.pageCount

  let firstPage = Math.max(data.page - 2, 1)
  let lastPage = Math.min(firstPage + 5, result.pageCount)
  result.pages = []
  for (let i = firstPage; i <= lastPage; i++) {
    result.pages.push(i)
  }

  return result
})

function loadPage(p) {
  emit('pageChange', p)
}
</script>

<template>
  <div class="card-footer d-flex align-items-center">
    <p class="m-0 text-muted">
      Showing <span>{{ pagination.start.toLocaleString() }}</span> to <span>{{ pagination.end.toLocaleString() }}</span> of
      <span>{{ data.count.toLocaleString() }}</span> entries
    </p>
    <ul class="pagination m-0 ms-auto" v-if="pagination.pageCount > 1">
      <li :class="'page-item' + (pagination.hasPrev ? '' : ' disabled')">
        <a class="page-link" @click.prevent="loadPage(data.page - 1)" :aria-disabled="pagination.hasPrev">
          <IconChevronLeft class="icon" />
          prev
        </a>
      </li>
      <li v-for="p in pagination.pages" :class="'page-item' + (p === data.page ? ' active' : '')">
        <a class="page-link" @click.prevent="loadPage(p)">{{ p }}</a>
      </li>
      <li :class="'page-item' + (pagination.hasNext ? '' : ' disabled')">
        <a class="page-link" @click.prevent="loadPage(data.page + 1)" :aria-disabled="!pagination.hasNext">
          next
          <IconChevronRight class="icon" />
        </a>
      </li>
    </ul>
  </div>
</template>

<style scoped></style>
