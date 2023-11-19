<script setup>
import { onMounted, reactive, ref } from 'vue'
import { IconFolders } from '@tabler/icons-vue'
import CardWithPlaceholder from '@/components/CardWithPlaceholder.vue'
import { filters } from '@/utils/filters'

const card = ref(null)

const data = reactive({
  text: '',
  details: '',
})

onMounted(function () {
  let f = filters.toQueryString()
  card.value.request(`/api/stats/count/projects?${f}`, function (response) {
    data.text = response.total.toLocaleString() + ' Projects'
    data.details = response.external.toLocaleString() + ' external'
  })
})
</script>

<template>
  <CardWithPlaceholder ref="card" type="count">
    <div class="card-body">
      <div class="row align-items-center">
        <div class="col-auto">
          <span class="bg-azure text-white avatar">
            <IconFolders />
          </span>
        </div>
        <div class="col">
          <div class="font-weight-medium">{{ data.text }}</div>
          <div class="text-muted">{{ data.details }}</div>
        </div>
      </div>
    </div>
  </CardWithPlaceholder>
</template>

<style scoped></style>
