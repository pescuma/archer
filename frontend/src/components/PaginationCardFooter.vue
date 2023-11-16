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
  result.hasFirst = data.page > 1
  result.hasLast = data.page < result.pageCount

  let firstPage = Math.max(Math.min(data.page - 2, result.pageCount - 4), 1)
  let lastPage = Math.min(firstPage + 4, result.pageCount)
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
  <div class="card-footer d-flex align-items-center" v-if="pagination.pageCount > 1">
    <p class="m-0 text-muted">
      Showing <span>{{ pagination.start.toLocaleString() }}</span> to <span>{{ pagination.end.toLocaleString() }}</span> of
      <span>{{ data.count.toLocaleString() }}</span> entries
    </p>
    <ul class="pagination m-0 ms-auto">
      <li :class="'page-item' + (pagination.hasFirst ? '' : ' disabled')">
        <a class="page-link" @click.prevent="loadPage(1)" :aria-disabled="pagination.hasFirst"> <IconChevronLeft class="icon" />first </a>
      </li>
      <li v-for="p in pagination.pages" :class="'page-item' + (p === data.page ? ' active' : '')">
        <a class="page-link" @click.prevent="loadPage(p)">{{ p }}</a>
      </li>
      <li :class="'page-item' + (pagination.hasLast ? '' : ' disabled')">
        <a class="page-link" @click.prevent="loadPage(pagination.pageCount)" :aria-disabled="!pagination.hasLast">
          last<IconChevronRight class="icon" />
        </a>
      </li>
    </ul>
  </div>
</template>

<style scoped></style>
